/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueReaction, IssueReactionCollection, ReactionCreate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export class IssueReactionService {
  constructor(private readonly api: ApiClient) {}
  async listIssueReactions(issueId: string): Promise<IssueReactionCollection> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/reactions", { params: { path: { issue_id: issueId } } })
    );
  }
  async createIssueReaction(issueId: string, body: ReactionCreate): Promise<IssueReaction> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/reactions", { params: { path: { issue_id: issueId } }, body })
    );
  }
  async deleteIssueReaction(issueId: string, reaction: string): Promise<void> {
    return unwrap(
      await this.api.DELETE("/api/v0/issues/{issue_id}/reactions/{reaction}", {
        params: { path: { issue_id: issueId, reaction } },
      })
    );
  }
}
