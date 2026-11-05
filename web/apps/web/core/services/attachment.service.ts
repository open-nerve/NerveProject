/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, AttachmentPage } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export type AttachmentTarget = { kind: "issue" | "comment" | "draft"; id: string };
export class AttachmentService {
  constructor(private readonly api: ApiClient) {}
  async list(target: AttachmentTarget, cursor?: string, signal?: AbortSignal): Promise<AttachmentPage> {
    const query = { limit: 50, cursor };
    switch (target.kind) {
      case "issue":
        return unwrap(
          await this.api.GET("/api/v0/issues/{issue_id}/attachments", {
            params: { path: { issue_id: target.id }, query },
            signal,
          })
        );
      case "comment":
        return unwrap(
          await this.api.GET("/api/v0/issue-comments/{comment_id}/attachments", {
            params: { path: { comment_id: target.id }, query },
            signal,
          })
        );
      case "draft":
        return unwrap(
          await this.api.GET("/api/v0/drafts/{draft_id}/attachments", {
            params: { path: { draft_id: target.id }, query },
            signal,
          })
        );
    }
  }
}
