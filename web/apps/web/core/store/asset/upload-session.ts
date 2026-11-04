/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import type { Asset, PreparedAsset } from "@nerve/api-client";
import { problemCode } from "@/lib/api-error";
import { putAsset } from "@/lib/asset-upload";
import type { AssetService } from "@/services/asset.service";
import type { RootStore } from "@/store/root.store";
import { ScopeLifetime, type AssetReadScope } from "./scope-lifetime";

export type ProfileUploadTarget =
  | { purpose: "USER_AVATAR" | "USER_COVER"; scope: { kind: "user" } }
  | { purpose: "WORKSPACE_LOGO"; scope: { kind: "workspace"; id: string } }
  | { purpose: "PROJECT_COVER"; scope: { kind: "project" | "new_project"; id: string } };
export type UploadPhase =
  | "idle"
  | "selected"
  | "preparing"
  | "uploading"
  | "confirming"
  | "ready"
  | "failed"
  | "cancelled"
  | "disposed";

export class UploadSession {
  phase: UploadPhase = "idle";
  error: unknown;
  readonly target: ProfileUploadTarget;
  private preview: string | undefined;
  private ready: Asset | undefined;
  private file: File | undefined;
  private prepared: PreparedAsset | undefined;
  private pending: Promise<Asset | undefined> | undefined;
  private controller: AbortController | undefined;
  private revision = 0;
  private disposed = false;
  private readonly lifetime: ScopeLifetime;

  constructor(
    private readonly root: RootStore,
    private readonly service: AssetService,
    target: ProfileUploadTarget,
    private readonly onDispose: () => void
  ) {
    const owned = { ...target };
    owned.scope = Object.freeze({ ...owned.scope });
    this.target = Object.freeze(owned);
    makeObservable<UploadSession, "preview" | "ready" | "disposed">(this, {
      phase: observable,
      error: observable.ref,
      preview: observable,
      ready: observable.ref,
      disposed: observable,
    });
    const scope = this.readScope();
    this.lifetime = new ScopeLifetime(
      root,
      scope,
      () => this.canWrite(scope),
      () => this.dispose()
    );
  }
  get current(): boolean {
    return !this.disposed && this.lifetime.current;
  }
  get busy(): boolean {
    return ["preparing", "uploading", "confirming"].includes(this.phase);
  }
  get previewURL(): string | undefined {
    return this.current ? this.preview : undefined;
  }
  get readyAsset(): Asset | undefined {
    return this.current ? this.ready : undefined;
  }
  get recovery(): "prepare" | "confirm" | "restart" | undefined {
    if (this.phase !== "failed" || !this.file) return undefined;
    return !this.prepared ? "prepare" : problemCode(this.error) === "asset.object_missing" ? "restart" : "confirm";
  }

  select(file: File): void {
    if (!this.current) return this.dispose();
    this.clear("idle");
    const limit = this.root.instance.config?.file_size_limit;
    const kind =
      !Number.isSafeInteger(limit) || !limit || limit < 1
        ? "unavailable"
        : !["image/jpeg", "image/png", "image/webp", "image/gif"].includes(file.type)
          ? "type"
          : file.size < 1 || file.size > limit
            ? "size"
            : undefined;
    runInAction(() => {
      if (kind) {
        this.error = new ImageSelectionError(kind);
        this.phase = "failed";
      } else {
        this.file = file;
        this.preview = URL.createObjectURL(file);
        this.phase = "selected";
      }
    });
  }

  start(): Promise<Asset | undefined> {
    if (!this.current) return Promise.resolve(undefined);
    if (this.pending) return this.pending;
    if (!this.file || (this.phase !== "selected" && this.recovery !== "prepare")) return this.wrongState();
    return this.begin("prepare");
  }
  retryComplete(): Promise<Asset | undefined> {
    if (!this.current) return Promise.resolve(undefined);
    if (this.pending) return this.pending;
    if (this.recovery !== "confirm") return this.wrongState();
    return this.begin("confirm");
  }
  restart(): Promise<Asset | undefined> {
    if (!this.current) return Promise.resolve(undefined);
    if (this.pending) return this.pending;
    if (!this.file || this.phase !== "failed") return this.wrongState();
    this.prepared = undefined;
    return this.begin("prepare");
  }
  private wrongState(): Promise<never> {
    return Promise.reject(new Error("Upload is not ready for this action"));
  }

  private begin(mode: "prepare" | "confirm"): Promise<Asset | undefined> {
    const revision = ++this.revision;
    const controller = new AbortController();
    this.controller = controller;
    // Publish the shared promise before changing observable phase (a synchronous observer may call start again).
    const pending = Promise.resolve()
      .then(() => this.perform(mode, revision, controller.signal))
      .finally(() => {
        if (this.pending === pending) {
          this.pending = undefined;
          this.controller = undefined;
        }
      });
    this.pending = pending;
    runInAction(() => {
      this.phase = mode === "prepare" ? "preparing" : "confirming";
      this.error = undefined;
    });
    return pending;
  }
  private alive(revision: number): boolean {
    if (!this.current) this.dispose();
    return revision === this.revision && this.current;
  }

  private async perform(
    mode: "prepare" | "confirm",
    revision: number,
    signal: AbortSignal
  ): Promise<Asset | undefined> {
    try {
      if (!this.alive(revision)) return undefined;
      if (mode === "prepare") {
        const file = this.file!;
        const prepared = await this.service.prepare(
          { ...this.target, name: file.name, size: file.size, content_type: file.type },
          signal
        );
        if (!this.alive(revision)) return undefined;
        this.prepared = prepared;
        runInAction(() => {
          this.phase = "uploading";
        });
        if (!this.alive(revision)) return undefined;
        await putAsset(file, prepared.upload, signal);
        if (!this.alive(revision)) return undefined;
      }
      runInAction(() => {
        this.phase = "confirming";
      });
      if (!this.alive(revision)) return undefined;
      const asset = await this.service.complete(this.prepared!.asset.id, signal);
      if (!this.alive(revision)) return undefined;
      if (asset.state !== "uploaded") throw new Error("Asset is not confirmed");
      runInAction(() => {
        this.ready = asset;
        this.phase = "ready";
      });
      return this.alive(revision) ? asset : undefined;
    } catch (error) {
      if (!this.alive(revision)) return undefined;
      runInAction(() => {
        this.error = error;
        this.phase = "failed";
      });
      if (!this.alive(revision)) return undefined;
      throw error;
    }
  }

  private readScope(): AssetReadScope {
    const { scope } = this.target;
    if (scope.kind === "user") return {};
    return scope.kind === "project"
      ? {
          workspaceID: this.root.projectRoot.project.getProjectById(scope.id)?.workspace_id,
          projectID: scope.id,
          member: true,
        }
      : { workspaceID: scope.id, member: true };
  }
  private canWrite(scope: AssetReadScope): boolean {
    if (this.target.scope.kind === "user") return true;
    const workspace = this.root.workspaceRoot.workspaces?.find((row) => row.id === scope.workspaceID);
    if (!workspace) return false;
    if (this.target.scope.kind === "workspace") return workspace.role === 20;
    if (this.target.scope.kind === "new_project") return workspace.role === 15 || workspace.role === 20;
    const project = this.root.projectRoot.project.getProjectById(scope.projectID);
    return (
      project?.workspace_id === workspace.id &&
      !project.archived_at &&
      this.root.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, scope.projectID) === 20
    );
  }
  private clear(phase: UploadPhase): void {
    ++this.revision;
    this.controller?.abort();
    this.controller = undefined;
    this.pending = undefined;
    if (this.preview) URL.revokeObjectURL(this.preview);
    this.file = undefined;
    this.prepared = undefined;
    runInAction(() => {
      this.preview = undefined;
      this.ready = undefined;
      this.error = undefined;
      this.phase = phase;
    });
  }
  cancel(): void {
    if (!this.disposed) this.clear("cancelled");
  }
  dispose(): void {
    if (this.disposed) return;
    runInAction(() => {
      this.disposed = true;
    });
    this.clear("disposed");
    this.lifetime.dispose();
    this.onDispose();
  }
}

export class ImageSelectionError extends Error {
  constructor(readonly kind: "type" | "size" | "unavailable") {
    super(`Image selection ${kind}`);
    this.name = "ImageSelectionError";
  }
}
