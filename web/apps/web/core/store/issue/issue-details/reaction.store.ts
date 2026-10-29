/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueReaction } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { upserted } from "@/lib/reconciled";
import type { IIssueDetail } from "./root.store";
import { IssueResources } from "./resources";

export type IIssueReactionStore = IssueReactionStore;
export type IIssueReactionStoreActions = Pick<
  IssueReactionStore,
  "fetchReactions" | "createReaction" | "removeReaction"
>;
export class IssueReactionStore {
  private readonly resources: IssueResources<IssueReaction[]>;
  constructor(readonly rootIssueDetailStore: IIssueDetail) {
    this.resources = new IssueResources(this.root);
  }
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }
  getReactionsByIssueId = (id: string) => {
    const rows = this.resources.get(id)?.value;
    if (!rows) return undefined;
    const groups: Record<string, string[]> = {};
    for (const row of rows) (groups[row.reaction] ??= []).push(row.id);
    return groups;
  };
  getReactionById = (id: string, reactionId: string) =>
    this.resources.get(id)?.value?.find((row) => row.id === reactionId);
  reactionsByUser = (id: string, actorId: string) =>
    (this.resources.get(id)?.value ?? []).filter((row) => row.actor_id === actorId);
  getError = (id: string) => this.resources.get(id)?.error;
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchReactions(ws, project, id);
  };
  private validate(rows: IssueReaction[], id: string) {
    if (rows.some((row) => row.issue_id !== id)) throw new SessionChangedError();
  }
  fetchReactions = async (ws: string, project: string, id: string): Promise<IssueReaction[]> => {
    const resource = this.resources.entry(ws, project, id);
    const value = await resource.fetch(async (read) => {
      const answer = await this.root.reactionApi.listIssueReactions(id);
      if (!read.isCurrent()) throw new SessionChangedError();
      this.validate(answer.data, id);
      return answer.data;
    });
    if (!value) throw new SessionChangedError();
    return value;
  };
  private write<T>(
    ws: string,
    project: string,
    id: string,
    send: () => Promise<T>,
    confirm: (answer: T, rows: IssueReaction[]) => IssueReaction[]
  ): Promise<T> {
    return this.root.enqueueIssueChanges([id], async () => {
      const resource = this.resources.entry(ws, project, id);
      const scope = this.root.scope(ws, project);
      const token = this.root.captureRead();
      const answer = await send();
      if (!this.root.isReadCurrent(token, [project], scope.workspaceID)) throw new SessionChangedError();
      this.root.assertProjectReadable(ws, project);
      resource.receive([], token, (held) => confirm(answer, held ?? []));
      await resource.refreshAfterConfirmation(async () => {
        await this.root.issues.refreshAuthoritative(id, { afterWrite: true });
        await this.rootIssueDetailStore.refreshAfterChange(ws, project, id);
      }, token);
      return answer;
    });
  }
  createReaction = (ws: string, project: string, id: string, reaction: string): Promise<IssueReaction> =>
    this.write(
      ws,
      project,
      id,
      () => this.root.reactionApi.createIssueReaction(id, { reaction }),
      (answer, held) => {
        this.validate([answer], id);
        if (answer.actor_id !== this.root.currentUserId || answer.reaction !== reaction)
          throw new SessionChangedError();
        return upserted(answer)(held);
      }
    );
  removeReaction = (ws: string, project: string, id: string, reaction: string): Promise<void> => {
    const actorId = this.root.currentUserId;
    if (!actorId) return Promise.reject(new SessionChangedError());
    return this.write(
      ws,
      project,
      id,
      () => this.root.reactionApi.deleteIssueReaction(id, reaction),
      (_, held) => held.filter((row) => row.actor_id !== actorId || row.reaction !== reaction)
    );
  };
}
