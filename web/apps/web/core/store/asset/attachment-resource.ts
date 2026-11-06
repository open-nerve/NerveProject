/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Asset, Attachment } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { AttachmentService, AttachmentTarget } from "@/services/attachment.service";
import type { AssetService } from "@/services/asset.service";
import type { IssueRootStore } from "@/store/issue/root.store";
import type { ContentView } from "./content-view";
import { PagedContent } from "./paged-content";
import { ContentDeletion } from "./content-deletion";
import { matchesContent } from "./content-target";
/** A confirmed command on a host's content: its kind and the asset it concerned. */
export type ContentConfirmation = { kind: "uploaded" | "deleted"; id: string };
export class AttachmentResource extends PagedContent<Attachment> {
  private readonly deletion: ContentDeletion;
  constructor(
    root: IssueRootStore,
    view: ContentView,
    list: AttachmentService,
    assets: AssetService,
    private readonly confirmed: (receipt: ContentConfirmation) => void = () => {},
    private readonly afterChange?: () => Promise<void>
  ) {
    const host = view.host;
    if (host.kind === "new_comment") throw new Error("A new comment has preparations, not an attachment list");
    const target: AttachmentTarget = {
      kind: host.kind,
      id: host.kind === "draft" ? host.draftID : host.kind === "comment" ? host.commentID : host.issueID,
    };
    super(root, view, (cursor, signal) => list.list(target, cursor, signal));
    this.deletion = new ContentDeletion(
      this,
      assets,
      (id) => confirmed({ kind: "deleted", id }),
      () => this.refresh()
    );
  }
  get uncertainIDs(): readonly string[] {
    return this.deletion.uncertainIDs;
  }
  remove(id: string): Promise<void> {
    return this.deletion.remove(id);
  }
  /** The read behind the refresh control also settles uncertain deletions. */
  async retryRead(): Promise<void> {
    await this.deletion.verify();
  }
  /** The uncertain IDs still visible after a fresh read; only those may be deleted again explicitly. */
  verifyDeletion(): Promise<ReadonlySet<string>> {
    return this.deletion.verify();
  }
  async afterUpload(asset: Asset): Promise<void> {
    this.assertCurrent();
    if (asset.state !== "uploaded" || !asset.is_bound || !matchesContent(asset, this.view.host, "attachment"))
      throw new SessionChangedError();
    this.confirmed({ kind: "uploaded", id: asset.id });
    if (this.current) await this.refreshConfirmed(() => this.refresh());
  }
  private async refresh(): Promise<void> {
    if (this.afterChange) await this.afterChange();
    else await this.load();
  }
  override dispose(): void {
    this.deletion?.dispose();
    super.dispose();
  }
}
