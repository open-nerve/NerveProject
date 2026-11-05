/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Asset } from "@nerve/api-client";
import type { AssetService } from "@/services/asset.service";
import type { IssueRootStore } from "@/store/issue/root.store";
import type { ContentView } from "./content-view";
import { PagedContent } from "./paged-content";
import { ContentDeletion } from "./content-deletion";
import { matchesContent } from "./content-target";
export class DraftPreparationsResource extends PagedContent<Asset> {
  private readonly deletion: ContentDeletion;
  constructor(root: IssueRootStore, view: ContentView, service: AssetService) {
    const host = view.host;
    if (host.kind !== "draft") throw new Error("Preparations belong to a private draft");
    super(root, view, async (cursor, signal) => {
      const page = await service.listDraftPreparations(host.draftID, cursor, signal);
      if (page.data.some((asset) => asset.is_bound || !matchesContent(asset, host)))
        throw new Error("Invalid draft preparation scope or binding");
      return page;
    });
    this.deletion = new ContentDeletion(
      this,
      service,
      () => {},
      () => this.load()
    );
  }
  get uncertainIDs(): readonly string[] {
    return this.deletion.uncertainIDs;
  }
  remove(id: string): Promise<void> {
    return this.deletion.remove(id);
  }
  /** The read behind the refresh control also settles uncertain deletions. */
  override async retryRead(): Promise<void> {
    await this.deletion.verify();
  }
  /** The uncertain IDs still visible after a fresh read; only those may be deleted again explicitly. */
  verifyDeletion(): Promise<ReadonlySet<string>> {
    return this.deletion.verify();
  }
  override dispose(): void {
    this.deletion?.dispose();
    super.dispose();
  }
}
