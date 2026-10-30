/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { CommentReaction, IssueComment, IssueCommentCreate } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { compareIssueTime } from "@/lib/issue-order";
import { upserted } from "@/lib/reconciled";
import type { IIssueDetail } from "./root.store";
import type { IssueReadToken } from "../root.store";
import { compareActivityOrder } from "./activity-order";
import { IssueResources } from "./resources";

type Comments = { data: IssueComment[]; cursor?: string; complete: boolean };
const sorted = (data: IssueComment[]) => data.toSorted(compareActivityOrder);
function merged(held: IssueComment[], answer: IssueComment[]): IssueComment[] {
  const rows = new Map(held.map((row) => [row.id, row]));
  for (const row of answer) {
    const previous = rows.get(row.id);
    if (!previous || compareIssueTime(row.updated_at, previous.updated_at) >= 0) rows.set(row.id, row);
  }
  return sorted([...rows.values()]);
}
export type IIssueCommentStore = IssueCommentStore;
export type IIssueCommentStoreActions = Pick<
  IssueCommentStore,
  "fetchComments" | "createComment" | "updateComment" | "removeComment"
>;

/** The comment is the sole owner of its embedded reactions; all commands share its issue's queue. */
export class IssueCommentStore {
  private readonly resources: IssueResources<Comments>;
  constructor(readonly rootIssueDetail: IIssueDetail) {
    this.resources = new IssueResources(this.root);
  }
  private get root() {
    return this.rootIssueDetail.rootIssueStore;
  }
  getCommentsByIssueId = (id: string) => this.resources.get(id)?.value?.data.map((row) => row.id);
  getCommentById = (id: string, commentId: string) =>
    this.resources.get(id)?.value?.data.find((row) => row.id === commentId);
  getError = (id: string) => this.resources.get(id)?.error;
  isLoading = (id: string) => this.resources.get(id)?.loading ?? false;
  refreshAfterChange = async (ws: string, project: string, id: string, verifyIssue = false): Promise<void> => {
    const resource = this.resources.get(id);
    if (!resource) return;
    await resource.refreshAfterConfirmation(async () => {
      if (verifyIssue || this.root.verification.isUncertain(id))
        await this.root.issues.refreshAuthoritative(id, { afterWrite: true });
      await this.fetchComments(ws, project, id, true);
    }, this.root.captureRead());
  };
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchComments(ws, project, id);
  };
  fetchComments = async (ws: string, project: string, id: string, incremental = false): Promise<IssueComment[]> => {
    const resource = this.resources.entry(ws, project, id);
    const before = resource.value;
    const value = await resource.fetch(async (read) => {
      const start = incremental && before?.complete ? before.cursor : undefined;
      let rows: IssueComment[] = [];
      const seen = new Set<string>();
      const pageFrom = async (cursor: string | undefined): Promise<Comments> => {
        const page = await this.root.commentApi.getIssueComments(id, cursor === undefined ? {} : { cursor });
        if (!read.isCurrent()) throw new SessionChangedError();
        this.validate(page.data, id);
        rows = merged(rows, page.data);
        if (page.next_cursor === null) {
          const retained =
            incremental && before?.complete
              ? before.data
              : (before?.data ?? []).filter((row) => rows.some((next) => next.id === row.id));
          return { data: merged(retained, rows), cursor, complete: true };
        }
        if (seen.has(page.next_cursor)) throw new Error("Repeated comment cursor");
        seen.add(page.next_cursor);
        return pageFrom(page.next_cursor);
      };
      return pageFrom(start);
    });
    if (!value) throw new SessionChangedError();
    return value.data;
  };
  private validate(rows: IssueComment[], id: string) {
    if (rows.some((row) => row.issue_id !== id || row.reactions.some((reaction) => reaction.comment_id !== row.id)))
      throw new SessionChangedError();
  }
  private current(token: IssueReadToken, ws: string, project: string) {
    const scope = this.root.scope(ws, project);
    if (!this.root.isReadCurrent(token, [project], scope.workspaceID)) throw new SessionChangedError();
    this.root.assertProjectReadable(ws, project);
  }
  private requireComment(id: string, commentId: string) {
    if (!this.getCommentById(id, commentId)) throw new Error("Comment has not been read");
  }
  private write<T>(
    ws: string,
    project: string,
    id: string,
    send: () => Promise<T>,
    confirm: (answer: T, value: Comments) => Comments,
    commentId?: string,
    changesIssue = true
  ): Promise<T> {
    return this.root.enqueueIssueChanges(
      [id],
      async () => {
        const resource = this.resources.entry(ws, project, id);
        if (commentId) this.requireComment(id, commentId);
        const token = this.root.captureRead();
        const answer = await send();
        this.current(token, ws, project);
        const initial: Comments = { data: [], complete: false };
        resource.receive(initial, token, (held) => confirm(answer, held ?? initial));
        await this.refreshAfterChange(ws, project, id, changesIssue);
        await this.rootIssueDetail.activity.refreshAfterChange(ws, project, id);
        return answer;
      },
      { workspaceSlug: ws, projectID: project }
    );
  }
  createComment = (ws: string, project: string, id: string, body: IssueCommentCreate): Promise<IssueComment> =>
    this.write(
      ws,
      project,
      id,
      () => this.root.commentApi.createIssueComment(id, { comment_html: body.comment_html }),
      (answer, held) => {
        this.validate([answer], id);
        return { ...held, data: merged(held.data, [answer]) };
      }
    );
  updateComment = (
    ws: string,
    project: string,
    id: string,
    commentId: string,
    body: IssueCommentCreate
  ): Promise<IssueComment> =>
    this.write(
      ws,
      project,
      id,
      () => this.root.commentApi.patchIssueComment(commentId, { comment_html: body.comment_html }),
      (answer, held) => {
        this.validate([answer], id);
        if (answer.id !== commentId) throw new SessionChangedError();
        return { ...held, data: merged(held.data, held.data.some((row) => row.id === commentId) ? [answer] : []) };
      },
      commentId
    );
  removeComment = (ws: string, project: string, id: string, commentId: string): Promise<void> =>
    this.write(
      ws,
      project,
      id,
      () => this.root.commentApi.deleteIssueComment(commentId),
      (_, held) => ({ ...held, data: held.data.filter((row) => row.id !== commentId) }),
      commentId
    );
  createReaction = (
    ws: string,
    project: string,
    id: string,
    commentId: string,
    reaction: string
  ): Promise<CommentReaction> =>
    this.write(
      ws,
      project,
      id,
      () => this.root.commentApi.addReaction(commentId, { reaction }),
      (answer, held) => {
        if (
          answer.comment_id !== commentId ||
          answer.actor_id !== this.root.currentUserId ||
          answer.reaction !== reaction
        )
          throw new SessionChangedError();
        return {
          ...held,
          data: held.data.map((row) =>
            row.id === commentId ? { ...row, reactions: upserted(answer)(row.reactions) } : row
          ),
        };
      },
      commentId,
      false
    );
  removeReaction = (ws: string, project: string, id: string, commentId: string, reaction: string): Promise<void> => {
    const actor = this.root.currentUserId;
    if (!actor) return Promise.reject(new SessionChangedError());
    return this.write(
      ws,
      project,
      id,
      () => this.root.commentApi.removeReaction(commentId, reaction),
      (_, held) => ({
        ...held,
        data: held.data.map((row) =>
          row.id === commentId
            ? {
                ...row,
                reactions: row.reactions.filter((value) => value.actor_id !== actor || value.reaction !== reaction),
              }
            : row
        ),
      }),
      commentId,
      false
    );
  };
}
