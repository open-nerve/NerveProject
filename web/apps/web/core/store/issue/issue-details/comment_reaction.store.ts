/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IIssueDetail } from "./root.store";

export type IIssueCommentReactionStore = IssueCommentReactionStore;
export type IIssueCommentReactionStoreActions = Pick<
  IssueCommentReactionStore,
  "createCommentReaction" | "removeCommentReaction"
>;
/** Presentation selectors only: the comment owns all reaction DTOs and confirmations. */
export class IssueCommentReactionStore {
  constructor(private readonly detail: IIssueDetail) {}
  private rows = (id: string, commentId: string) => this.detail.comment.getCommentById(id, commentId)?.reactions;
  getCommentReactionsByCommentId = (id: string, commentId: string) => {
    const rows = this.rows(id, commentId);
    if (!rows) return undefined;
    const groups: Record<string, string[]> = {};
    for (const row of rows) (groups[row.reaction] ??= []).push(row.id);
    return groups;
  };
  getCommentReactionById = (id: string, commentId: string, reactionId: string) =>
    this.rows(id, commentId)?.find((row) => row.id === reactionId);
  commentReactionsByUser = (id: string, commentId: string, userId: string) =>
    (this.rows(id, commentId) ?? []).filter((row) => row.actor_id === userId);
  createCommentReaction = (ws: string, project: string, id: string, commentId: string, reaction: string) =>
    this.detail.comment.createReaction(ws, project, id, commentId, reaction);
  removeCommentReaction = (ws: string, project: string, id: string, commentId: string, reaction: string) =>
    this.detail.comment.removeReaction(ws, project, id, commentId, reaction);
}
