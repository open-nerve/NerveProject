/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, Asset } from "@nerve/api-client";
import { AttachmentService } from "@/services/attachment.service";
import { AssetService } from "@/services/asset.service";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { RootStore } from "@/store/root.store";
import { CommentPreparations } from "./comment-preparations";
import { contentTarget, matchesContent } from "./content-target";
import { AttachmentResource, type ContentConfirmation } from "./attachment-resource";
import { ContentReads } from "./content-reads";
import { DraftPreparationsResource } from "./draft-preparations-resource";
import { UploadSession } from "./upload-session";
import { contentUploadPolicy, profileUploadPolicy, type ProfileUploadTarget } from "./upload-policy";
import { ProfileImages } from "./profile-images";
import { ContentView, type ContentHost, type ContentUse } from "./content-view";

export class RootAssetStore {
  readonly images: ProfileImages;
  private readonly service: AssetService;
  private readonly attachmentService: AttachmentService;
  private readonly reads = new WeakMap<ContentView, ContentReads>();
  private readonly children = new Set<{ dispose(): void }>();
  private readonly uploads = new Set<UploadSession>();
  private readonly views = new Set<ContentView>();
  private disposed = false;
  constructor(
    private readonly root: RootStore,
    api: ApiClient
  ) {
    this.service = new AssetService(api);
    this.attachmentService = new AttachmentService(api);
    this.images = new ProfileImages(root);
  }
  createUpload(target: ProfileUploadTarget): UploadSession {
    this.root.issue.assertActive();
    if (this.disposed) throw new SessionChangedError();
    const upload = new UploadSession(this.service, profileUploadPolicy(this.root, target), () =>
      this.uploads.delete(upload)
    );
    this.uploads.add(upload);
    return upload;
  }
  openContent(host: ContentHost): ContentView {
    this.root.issue.assertActive();
    if (this.disposed) throw new SessionChangedError();
    const view = new ContentView(this.root, host);
    if (view.current) {
      this.views.add(view);
      view.signal.addEventListener("abort", () => this.views.delete(view), { once: true });
    }
    return view;
  }
  createContentUpload(view: ContentView, use: ContentUse): UploadSession {
    this.root.issue.assertActive();
    if (this.disposed || !this.views.has(view) || !view.canWrite(use)) throw new SessionChangedError();
    const upload = new UploadSession(this.service, contentUploadPolicy(this.root, view, use), () =>
      this.uploads.delete(upload)
    );
    this.uploads.add(upload);
    return upload;
  }
  private own<T extends { dispose(): void }>(view: ContentView, create: () => T): T {
    this.root.issue.assertActive();
    if (this.disposed || !this.views.has(view) || !view.current) throw new SessionChangedError();
    const child = create();
    this.children.add(child);
    view.signal.addEventListener(
      "abort",
      () => {
        child.dispose();
        this.children.delete(child);
      },
      { once: true }
    );
    return child;
  }
  async inspect(view: ContentView, id: string, use: ContentUse, signal?: AbortSignal): Promise<Asset> {
    if (this.disposed || !this.views.has(view) || !view.current) throw new SessionChangedError();
    const asset = await this.service.read(id, signal ?? view.signal);
    if (!view.current) throw new SessionChangedError();
    if (asset.id !== id || asset.state !== "uploaded" || !matchesContent(asset, view.host, use))
      throw new Error("Invalid content asset");
    return asset;
  }
  commentPreparations(
    view: ContentView,
    ids: readonly string[],
    changed: (ids: string[]) => void
  ): CommentPreparations {
    return this.own(
      view,
      () => new CommentPreparations(view, (id, signal) => this.inspect(view, id, "attachment", signal), ids, changed)
    );
  }
  async duplicateImage(view: ContentView, sourceID: string, signal: AbortSignal): Promise<Asset> {
    this.root.issue.assertActive();
    if (this.disposed || !this.views.has(view) || !view.canWrite("description") || signal.aborted)
      throw new SessionChangedError();
    const asset = await this.service.duplicate(sourceID, contentTarget(view.host, "description"), signal);
    if (!view.canWrite("description") || signal.aborted) throw new SessionChangedError();
    if (asset.state !== "uploaded" || !matchesContent(asset, view.host, "description"))
      throw new Error("Invalid duplicated image");
    return asset;
  }
  attachments(view: ContentView, confirmed?: (receipt: ContentConfirmation) => void): AttachmentResource {
    const host = view.host;
    const afterChange =
      host.kind !== "issue"
        ? undefined
        : async () => {
            const issue = this.root.issue;
            const token = issue.captureRead();
            const workspace = this.root.workspaceRoot.workspaces?.find((row) => row.id === host.workspaceID);
            if (!workspace) throw new SessionChangedError();
            await issue.issues.refreshAuthoritative(host.issueID, { afterWrite: true });
            if (!issue.isReadCurrent(token, [host.projectID], host.workspaceID)) return;
            if (!issue.issues.getIssueDetailById(host.issueID)) return;
            await issue.issueDetail.refreshAfterChange(workspace.slug, host.projectID, host.issueID);
          };
    return this.own(
      view,
      () => new AttachmentResource(this.root.issue, view, this.attachmentService, this.service, confirmed, afterChange)
    );
  }
  content(view: ContentView): ContentReads {
    const existing = this.reads.get(view);
    if (existing) return existing;
    const reads = this.own(view, () => new ContentReads(view, this.service));
    this.reads.set(view, reads);
    return reads;
  }
  draftPreparations(view: ContentView): DraftPreparationsResource {
    return this.own(view, () => new DraftPreparationsResource(this.root.issue, view, this.service));
  }
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    for (const view of this.views) view.dispose();
    for (const child of this.children) child.dispose();
    this.children.clear();
    for (const upload of this.uploads) upload.dispose();
    this.images.dispose();
  }
}
