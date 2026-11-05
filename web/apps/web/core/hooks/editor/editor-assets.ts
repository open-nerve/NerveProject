/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { reaction } from "mobx";
import type { Asset } from "@nerve/api-client";
import type { EditorImages, ImageOperation, ImageTransferPhase } from "@nerve/editor";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { downloadAsset } from "@/lib/asset-download";
import type { RootStore } from "@/store/root.store";
import type { ContentView } from "@/store/asset/content-view";
import type { UploadSession } from "@/store/asset/upload-session";
import { ImageQueue } from "./image-queue";
import { ImageRead, isContentImage } from "./image-read";
type NodeTransfer = { controller?: AbortController; upload?: UploadSession; pending?: Promise<string> };
/** One mounted editor owns its operations; RootAssetStore continues to own protocol and authorization. */
export class EditorAssets {
  readonly images: EditorImages;
  private readonly lifetime = new AbortController();
  private readonly queue = new ImageQueue();
  private readonly nodes = new Map<string, NodeTransfer>();
  private readonly reads = new Set<ImageRead>();
  constructor(
    private readonly root: RootStore,
    private readonly view: ContentView,
    private readonly message: (error: unknown) => string
  ) {
    const maxBytes = root.instance.config?.file_size_limit;
    this.images = {
      signal: this.lifetime.signal,
      read: (id) => {
        this.assertCurrent();
        const lease = root.asset.content(view).acquire(id);
        const read = new ImageRead(lease, view.host, message, () => this.reads.delete(read));
        this.reads.add(read);
        return read;
      },
      download: (id, signal) => this.download(id, signal),
      ...(view.canWrite("description") && Number.isSafeInteger(maxBytes) && maxBytes! > 0
        ? {
            write: {
              upload: (nodeID: string, file: File, operation: ImageOperation) => this.upload(nodeID, file, operation),
              retryUpload: (nodeID: string, operation: ImageOperation) => this.retry(nodeID, operation),
              duplicate: (nodeID: string, sourceID: string, operation: ImageOperation) =>
                this.duplicate(nodeID, sourceID, operation),
              cancel: (nodeID: string) => this.cancel(nodeID),
              get maxBytes() {
                return root.instance.config?.file_size_limit ?? 0;
              },
            },
          }
        : {}),
    };
    view.signal.addEventListener("abort", this.dispose, { once: true });
    if (!view.current) this.dispose();
  }
  private assertCurrent() {
    if (this.lifetime.signal.aborted || !this.view.current) throw new SessionChangedError();
  }
  private async download(id: string, signal: AbortSignal): Promise<void> {
    this.assertCurrent();
    const controller = new AbortController();
    const abort = () => controller.abort();
    signal.addEventListener("abort", abort, { once: true });
    this.lifetime.signal.addEventListener("abort", abort, { once: true });
    if (signal.aborted || this.lifetime.signal.aborted) abort();
    try {
      const read = await this.root.asset.content(this.view).read(id, controller.signal);
      this.assertCurrent();
      if (controller.signal.aborted) throw new SessionChangedError();
      if (!isContentImage({ status: "ready", stamp: 0, ...read }, this.view.host)) throw new Error("Invalid image");
      await downloadAsset(read, controller.signal);
      this.assertCurrent();
    } catch (error) {
      throw error instanceof SessionChangedError ? error : new Error(this.message(error));
    } finally {
      signal.removeEventListener("abort", abort);
      this.lifetime.signal.removeEventListener("abort", abort);
    }
  }
  private async upload(id: string, file: File, operation: ImageOperation): Promise<string> {
    this.cancel(id);
    this.assertCurrent();
    const upload = this.root.asset.createContentUpload(this.view, "description");
    const node: NodeTransfer = { upload };
    this.nodes.set(id, node);
    upload.select(file);
    return this.perform(id, node, operation, () => {
      if (upload.phase === "failed") throw upload.error;
      return upload.start();
    });
  }
  private async retry(id: string, operation: ImageOperation): Promise<string> {
    this.assertCurrent();
    const node = this.nodes.get(id);
    if (node?.pending) return node.pending;
    if (!node?.upload || node.upload.phase !== "failed") throw new Error(this.message(undefined));
    const upload = node.upload;
    return this.perform(id, node, operation, () =>
      upload.recovery === "confirm"
        ? upload.retryComplete()
        : upload.recovery === "restart"
          ? upload.restart()
          : upload.start()
    );
  }
  private async duplicate(id: string, sourceID: string, operation: ImageOperation): Promise<string> {
    this.cancel(id);
    this.assertCurrent();
    const node: NodeTransfer = {};
    this.nodes.set(id, node);
    return this.perform(id, node, operation, async (signal) => {
      operation.onPhase("preparing");
      this.checkNode(id, node, signal);
      return this.root.asset.duplicateImage(this.view, sourceID, signal);
    });
  }
  private checkNode(id: string, node: NodeTransfer, signal: AbortSignal) {
    this.assertCurrent();
    if (this.nodes.get(id) !== node || signal.aborted || !this.view.canWrite("description"))
      throw new SessionChangedError();
  }
  private perform(
    id: string,
    node: NodeTransfer,
    operation: ImageOperation,
    work: (signal: AbortSignal) => Promise<Asset | undefined>
  ): Promise<string> {
    const controller = new AbortController();
    node.controller = controller;
    const abort = () => {
      if (this.nodes.get(id) === node) this.cancel(id);
      else controller.abort();
    };
    operation.signal.addEventListener("abort", abort, { once: true });
    if (operation.signal.aborted) abort();
    let stop: (() => void) | undefined;
    const pending = Promise.resolve()
      .then(async () => {
        this.checkNode(id, node, controller.signal);
        operation.onPhase("queued");
        this.checkNode(id, node, controller.signal);
        return this.queue.run(controller.signal, async () => {
          this.checkNode(id, node, controller.signal);
          if (node.upload)
            stop = reaction(
              () => node.upload!.phase,
              (phase) => {
                if (["preparing", "uploading", "confirming"].includes(phase) && !controller.signal.aborted)
                  operation.onPhase(phase as ImageTransferPhase);
              }
            );
          const asset = await work(controller.signal);
          this.checkNode(id, node, controller.signal);
          if (!asset) throw new SessionChangedError();
          return asset.id;
        });
      })
      .then((value) => {
        this.checkNode(id, node, controller.signal);
        this.nodes.delete(id);
        node.upload?.dispose();
        return value;
      })
      .catch((error: unknown) => {
        if (!node.upload && this.nodes.get(id) === node) this.nodes.delete(id);
        throw error instanceof SessionChangedError ? error : new Error(this.message(error));
      })
      .finally(() => {
        stop?.();
        operation.signal.removeEventListener("abort", abort);
        if (node.pending === pending) node.pending = undefined;
      });
    node.pending = pending;
    return pending;
  }
  private cancel(id: string) {
    const node = this.nodes.get(id);
    if (!node) return;
    this.nodes.delete(id);
    node.controller?.abort();
    node.upload?.dispose();
  }
  dispose = () => {
    if (this.lifetime.signal.aborted) return;
    this.lifetime.abort();
    for (const id of this.nodes.keys()) this.cancel(id);
    for (const read of this.reads) read.dispose();
    this.view.signal.removeEventListener("abort", this.dispose);
  };
}
