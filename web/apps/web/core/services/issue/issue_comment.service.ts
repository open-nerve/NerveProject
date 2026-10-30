/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type {
  ApiClient,
  CommentReaction,
  IssueComment,
  IssueCommentCreate,
  IssueCommentPage,
  operations,
  ReactionCreate,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export class IssueCommentService {
  constructor(private readonly api: ApiClient) {}
  async getIssueComments(
    issueId: string,
    query: operations["listIssueComments"]["parameters"]["query"] = {}
  ): Promise<IssueCommentPage> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/comments", { params: { path: { issue_id: issueId }, query } })
    );
  }
  async createIssueComment(issueId: string, body: IssueCommentCreate): Promise<IssueComment> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/comments", { params: { path: { issue_id: issueId } }, body })
    );
  }
  async patchIssueComment(commentId: string, body: IssueCommentCreate): Promise<IssueComment> {
    return unwrap(
      await this.api.PATCH("/api/v0/issue-comments/{comment_id}", { params: { path: { comment_id: commentId } }, body })
    );
  }
  async deleteIssueComment(commentId: string): Promise<void> {
    return unwrap(
      await this.api.DELETE("/api/v0/issue-comments/{comment_id}", { params: { path: { comment_id: commentId } } })
    );
  }
  async addReaction(commentId: string, body: ReactionCreate): Promise<CommentReaction> {
    return unwrap(
      await this.api.POST("/api/v0/issue-comments/{comment_id}/reactions", {
        params: { path: { comment_id: commentId } },
        body,
      })
    );
  }
  async removeReaction(commentId: string, reaction: string): Promise<void> {
    return unwrap(
      await this.api.DELETE("/api/v0/issue-comments/{comment_id}/reactions/{reaction}", {
        params: { path: { comment_id: commentId, reaction } },
      })
    );
  }
}
