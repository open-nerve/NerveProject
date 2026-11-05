/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Editor } from "@tiptap/core";
import { closeHistory } from "@tiptap/pm/history";
import type { EditorImages, ImageOperation, ImageTransferPhase } from "@/types";
import { isImageID } from "./input";
import { ECustomImageStatus } from "./types";

export type ImageTransferState = {
  phase?: ImageTransferPhase;
  error?: string;
  failed: boolean;
  copied: boolean;
  /** The node left the document while its transfer ran; the same node brought back (Redo) starts nothing by itself. */
  cancelled: boolean;
};

const idle: ImageTransferState = { failed: false, copied: false, cancelled: false };

/** Positions of the node a transfer belongs to: the image with this id that has no src yet. */
function waiting(editor: Editor, id: string): number[] {
  const found: number[] = [];
  editor.state.doc.descendants((node, pos) => {
    if (node.type.name === "imageComponent" && node.attrs.id === id && !node.attrs.src) found.push(pos);
  });
  return found;
}

/**
 * Image transfers belong to the node with that id and no src in the document, not to a node view: a view rebuilt for
 * the same node (a block dragged far, wrapped or unwrapped) shows the transfer that is already running. A transfer
 * ends when its node leaves the document, the editor or the image capability ends, or the user cancels it.
 */
export class ImageTransfers {
  private readonly states = new Map<string, ImageTransferState>();
  private readonly running = new Map<string, AbortController>();
  private readonly listeners = new Map<string, Set<() => void>>();

  constructor(
    private readonly images: EditorImages | undefined,
    private readonly settled: { fileMap: Map<string, unknown>; completed: Map<string, string> }
  ) {
    images?.signal.addEventListener("abort", () => this.destroy(), { once: true });
  }

  state(id: string): ImageTransferState {
    return this.states.get(id) ?? idle;
  }

  subscribe(id: string, listener: () => void): () => void {
    let set = this.listeners.get(id);
    if (!set) this.listeners.set(id, (set = new Set()));
    set.add(listener);
    return () => {
      set.delete(listener);
      if (!set.size) this.listeners.delete(id);
    };
  }

  /** Starts a transfer for the node, ending any earlier one for the same id. */
  start(editor: Editor, id: string, copied: boolean, run: (operation: ImageOperation) => Promise<string>): void {
    this.running.get(id)?.abort();
    const controller = new AbortController();
    this.running.set(id, controller);
    this.set(id, { phase: "queued", failed: false, copied, cancelled: false });
    const current = () =>
      this.running.get(id) === controller &&
      !controller.signal.aborted &&
      !editor.isDestroyed &&
      editor.isEditable &&
      !!this.images &&
      !this.images.signal.aborted &&
      waiting(editor, id).length > 0;
    void Promise.resolve()
      .then(() => {
        if (!current()) throw new Error("Cancelled");
        return run({
          signal: controller.signal,
          onPhase: (phase) => {
            if (current()) this.set(id, { ...this.state(id), phase });
          },
        });
      })
      .then((src) => {
        if (!current()) return;
        if (!isImageID(src)) throw new Error("Invalid image identity");
        this.running.delete(id);
        this.states.delete(id);
        this.settled.fileMap.delete(id);
        this.settled.completed.set(id, src);
        const tr = editor.state.tr;
        for (const pos of waiting(editor, id))
          tr.setNodeMarkup(pos, undefined, {
            ...editor.state.doc.nodeAt(pos)!.attrs,
            src,
            status: ECustomImageStatus.UPLOADED,
          });
        editor.view.dispatch(tr.setMeta("addToHistory", false));
        if (!editor.isDestroyed) editor.view.dispatch(closeHistory(editor.state.tr));
        this.notify(id);
        return undefined;
      })
      .catch((cause: unknown) => {
        if (!current()) return;
        this.running.delete(id);
        this.set(id, {
          failed: true,
          copied,
          cancelled: false,
          error: cause instanceof Error ? cause.message : undefined,
        });
      });
  }

  /** The user's cancel: the transfer ends and the node waits for a new choice. */
  cancel(id: string): void {
    this.running.get(id)?.abort();
    this.running.delete(id);
    this.images?.write?.cancel(id);
    this.set(id, idle);
  }

  /**
   * A view of the node is gone. Its transfer ends only if, when the task ends, the node is no longer in the document;
   * a view rebuilt for the same node in that task keeps it.
   */
  release(editor: Editor, id: string): void {
    queueMicrotask(() => {
      const controller = this.running.get(id);
      if (!controller || (!editor.isDestroyed && waiting(editor, id).length > 0)) return;
      controller.abort();
      this.running.delete(id);
      this.images?.write?.cancel(id);
      this.set(id, { ...idle, cancelled: true });
    });
  }

  /** The editor or the image capability ended: every transfer ends with it. */
  destroy(): void {
    for (const [id, controller] of this.running) {
      controller.abort();
      this.images?.write?.cancel(id);
    }
    this.running.clear();
    this.states.clear();
    for (const id of this.listeners.keys()) this.notify(id);
  }

  private set(id: string, state: ImageTransferState): void {
    this.states.set(id, state);
    this.notify(id);
  }

  private notify(id: string): void {
    for (const listener of this.listeners.get(id) ?? []) listener();
  }
}
