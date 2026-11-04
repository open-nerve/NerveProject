/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient } from "@nerve/api-client";
import { AssetService } from "@/services/asset.service";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { RootStore } from "@/store/root.store";
import { UploadSession } from "./upload-session";
import { contentUploadPolicy, profileUploadPolicy, type ProfileUploadTarget } from "./upload-policy";
import { ProfileImages } from "./profile-images";
import { ContentView, type ContentHost, type ContentUse } from "./content-view";

export class RootAssetStore {
  readonly images: ProfileImages;
  private readonly service: AssetService;
  private readonly uploads = new Set<UploadSession>();
  private readonly views = new Set<ContentView>();
  private disposed = false;
  constructor(
    private readonly root: RootStore,
    api: ApiClient
  ) {
    this.service = new AssetService(api);
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
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    for (const view of this.views) view.dispose();
    for (const upload of this.uploads) upload.dispose();
    this.images.dispose();
  }
}
