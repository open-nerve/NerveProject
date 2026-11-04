/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient } from "@nerve/api-client";
import { AssetService } from "@/services/asset.service";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { RootStore } from "@/store/root.store";
import { UploadSession, type ProfileUploadTarget } from "./upload-session";
import { ProfileImages } from "./profile-images";

export class RootAssetStore {
  readonly images: ProfileImages;
  private readonly service: AssetService;
  private readonly uploads = new Set<UploadSession>();
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
    const upload = new UploadSession(this.root, this.service, target, () => this.uploads.delete(upload));
    this.uploads.add(upload);
    return upload;
  }
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    for (const upload of this.uploads) upload.dispose();
    this.images.dispose();
  }
}
