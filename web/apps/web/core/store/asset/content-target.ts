/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Asset, NewAsset } from "@nerve/api-client";
import type { ContentHost, ContentUse } from "./content-view";
export function contentTarget(host: ContentHost, use: ContentUse): Pick<NewAsset, "purpose" | "scope"> {
  const purpose: NewAsset["purpose"] =
    host.kind === "draft"
      ? use === "attachment"
        ? "DRAFT_ISSUE_ATTACHMENT"
        : "DRAFT_ISSUE_DESCRIPTION"
      : host.kind === "issue"
        ? use === "attachment"
          ? "ISSUE_ATTACHMENT"
          : "ISSUE_DESCRIPTION"
        : use === "attachment"
          ? "COMMENT_ATTACHMENT"
          : "COMMENT_DESCRIPTION";
  const id = host.kind === "draft" ? host.draftID : host.kind === "comment" ? host.commentID : host.issueID;
  return { purpose, scope: { kind: host.kind, id } };
}
export function matchesContent(asset: Asset, host: ContentHost, use?: ContentUse): boolean {
  return (use ? [use] : (["attachment", "description"] as const)).some((value) => {
    const target = contentTarget(host, value);
    return (
      asset.purpose === target.purpose && asset.scope.kind === target.scope.kind && asset.scope.id === target.scope.id
    );
  });
}
