/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Draft, DraftCreate, DraftUpdate, IssueDetail } from "@nerve/api-client";
import { observable, runInAction } from "mobx";
import type { Group } from "@/lib/paged-groups";
import type { TLoader, ViewFlags } from "@nerve/types";
import { oneAtATime } from "@/lib/one-at-a-time";
import { compareCodePoints, compareIssueTime } from "@/lib/issue-order";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { DetailResource } from "../issue-details/resource";
import type { IssueReadToken, IssueRootStore } from "../root.store";

type Drafts = { data: Draft[]; total?: number; next_cursor?: string | null; listed: boolean };
type DraftInput = DraftUpdate | ((held: Draft) => DraftUpdate);
const empty = (): Drafts => ({ data: [], listed: false });
function merged(before: Draft[], incoming: Draft[]): Draft[] {
  const rows = new Map(before.map((row) => [row.id, row]));
  for (const row of incoming) {
    const old = rows.get(row.id);
    if (!old || compareIssueTime(old.updated_at, row.updated_at) <= 0) rows.set(row.id, row);
  }
  return [...rows.values()].toSorted(
    (a, b) => compareIssueTime(b.created_at, a.created_at) || compareCodePoints(b.id, a.id)
  );
}
export type IWorkspaceDraftIssues = WorkspaceDraftIssues;
export class WorkspaceDraftIssues {
  private readonly resources = observable.map<string, DetailResource<Drafts>>({}, { deep: false });
  private readonly reads = observable.map<string, "init-loader" | "pagination">();
  private readonly mutations = observable.map<symbol, { workspaceID: string; token: IssueReadToken }>(
    {},
    { deep: false }
  );
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();
  constructor(readonly issueStore: IssueRootStore) {}
  canRead = (ws: string): boolean => {
    const workspace = this.issueStore.rootStore.workspaceRoot.getWorkspaceBySlug(ws);
    return this.issueStore.active && !!workspace && (workspace.role === 15 || workspace.role === 20);
  };
  private held(ws: string | undefined) {
    const workspace = ws && this.issueStore.rootStore.workspaceRoot.getWorkspaceBySlug(ws);
    return workspace && this.canRead(ws!) ? this.resources.get(workspace.id) : undefined;
  }
  private entry(ws: string): DetailResource<Drafts> {
    if (!this.canRead(ws)) throw new SessionChangedError();
    const workspace = this.issueStore.rootStore.workspaceRoot.getWorkspaceBySlug(ws)!;
    let resource = this.resources.get(workspace.id);
    if (!resource) {
      resource = new DetailResource(this.issueStore, { workspaceID: workspace.id, projectIDs: [], member: true });
      runInAction(() => this.resources.set(workspace.id, resource!));
    }
    return resource;
  }
  getDrafts = (ws: string) => this.held(ws)?.value?.data;
  getTotal = (ws: string) => this.held(ws)?.value?.total;
  hasMore = (ws: string) => this.held(ws)?.value?.next_cursor != null;
  getError = (ws: string) => this.held(ws)?.error;
  getIssueById = (id: string, ws = this.issueStore.workspaceSlug) =>
    this.held(ws)?.value?.data.find((row) => row.id === id);
  get issueIds(): string[] {
    return this.held(this.issueStore.workspaceSlug)?.value?.data.map((row) => row.id) ?? [];
  }
  getLoader = (ws: string) => {
    const resource = this.held(ws);
    if (!resource) return undefined;
    if (resource.loading) return this.reads.get(resource.scope.workspaceID);
    return [...this.mutations.values()].some(
      (row) =>
        row.workspaceID === resource.scope.workspaceID && this.issueStore.isReadCurrent(row.token, [], row.workspaceID)
    )
      ? "mutation"
      : undefined;
  };
  private assertCurrent(ws: string, resource: DetailResource<Drafts>, token: IssueReadToken) {
    if (!this.canRead(ws) || !this.issueStore.isReadCurrent(token, [], resource.scope.workspaceID))
      throw new SessionChangedError();
  }
  private validate(rows: Draft[], resource: DetailResource<Drafts>, id?: string) {
    if (rows.some((row) => row.workspace_id !== resource.scope.workspaceID || (id !== undefined && row.id !== id)))
      throw new SessionChangedError();
  }
  fetchIssues = async (ws: string, next = false): Promise<Draft[]> => {
    const resource = this.entry(ws);
    const before = resource.value;
    if (next && before?.next_cursor === null) return before.data;
    const cursor = next ? (before?.next_cursor ?? undefined) : undefined;
    runInAction(() => this.reads.set(resource.scope.workspaceID, next ? "pagination" : "init-loader"));
    const answer = await resource.fetch(async (read) => {
      const page = await this.issueStore.draftApi.list(ws, cursor === undefined ? {} : { cursor });
      if (!read.isCurrent()) throw new SessionChangedError();
      this.validate(page.data, resource);
      if (cursor !== undefined && cursor === page.next_cursor) throw new Error("Repeated draft cursor");
      return { ...page, data: merged(next ? (before?.data ?? []) : [], page.data), listed: true };
    });
    if (!answer) throw new SessionChangedError();
    return answer.data;
  };
  fetchIssue = async (ws: string, id: string): Promise<Draft | undefined> => {
    const resource = this.entry(ws);
    const before = resource.value ?? empty();
    const answer = await resource.fetch(async () => {
      const row = await this.issueStore.draftApi.get(id);
      this.validate([row], resource, id);
      return { ...before, data: merged(before.data, [row]) };
    });
    return answer?.data.find((row) => row.id === id);
  };
  retry = async (ws: string): Promise<void> => {
    const resource = this.entry(ws);
    if (!(await resource.retryConfirmationRead())) await this.fetchIssues(ws);
  };
  private async command<T>(
    ws: string,
    id: string | undefined,
    run: (resource: DetailResource<Drafts>, token: IssueReadToken) => Promise<T>
  ): Promise<T> {
    const resource = this.entry(ws);
    const token = this.issueStore.captureRead(true);
    const key = id ?? `new:${resource.scope.workspaceID}`;
    let queue = this.queues.get(key);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(key, queue);
    }
    return queue(async () => {
      this.assertCurrent(ws, resource, token);
      const operation = Symbol("draft mutation");
      runInAction(() => this.mutations.set(operation, { workspaceID: resource.scope.workspaceID, token }));
      try {
        return await run(resource, token);
      } finally {
        runInAction(() => this.mutations.delete(operation));
      }
    });
  }
  private remember(resource: DetailResource<Drafts>, token: IssueReadToken, row: Draft) {
    resource.receive(empty(), token, (held) => ({ ...(held ?? empty()), data: merged(held?.data ?? [], [row]) }));
  }
  private async refreshTotal(ws: string, resource: DetailResource<Drafts>, token: IssueReadToken) {
    if (resource.value?.listed) await resource.refreshAfterConfirmation(() => this.fetchIssues(ws), token);
  }
  createIssue = (ws: string, input: DraftCreate): Promise<Draft> =>
    this.command(ws, undefined, async (resource, token) => {
      const row = await this.issueStore.draftApi.create(ws, input);
      this.assertCurrent(ws, resource, token);
      this.validate([row], resource);
      this.remember(resource, token, row);
      resource.confirm((value) => ({ ...value, total: undefined }), token);
      await this.refreshTotal(ws, resource, token);
      return row;
    });
  private async save(
    ws: string,
    id: string,
    input: DraftInput,
    resource: DetailResource<Drafts>,
    token: IssueReadToken
  ): Promise<Draft> {
    const held = this.getIssueById(id, ws);
    if (!held) throw new Error("Draft has not been read");
    const body = typeof input === "function" ? input(held) : input;
    const row = await this.issueStore.draftApi.update(id, body);
    this.assertCurrent(ws, resource, token);
    this.validate([row], resource, id);
    this.remember(resource, token, row);
    return row;
  }
  updateIssue = (ws: string, id: string, input: DraftInput): Promise<Draft> =>
    this.command(ws, id, (resource, token) => this.save(ws, id, input, resource, token));
  private remove(resource: DetailResource<Drafts>, token: IssueReadToken, id: string) {
    resource.receive(empty(), token, (held) => ({
      ...(held ?? empty()),
      data: (held?.data ?? []).filter((row) => row.id !== id),
      total: undefined,
    }));
  }
  deleteIssue = (ws: string, id: string): Promise<void> =>
    this.command(ws, id, async (resource, token) => {
      if (!this.getIssueById(id, ws)) throw new Error("Draft has not been read");
      await this.issueStore.draftApi.delete(id);
      this.assertCurrent(ws, resource, token);
      this.remove(resource, token, id);
      await this.refreshTotal(ws, resource, token);
    });
  moveIssue = (ws: string, id: string, input: DraftInput): Promise<IssueDetail> =>
    this.command(ws, id, async (resource, token) => {
      await this.save(ws, id, input, resource, token);
      const scope = { workspaceSlug: ws, workspaceID: resource.scope.workspaceID };
      const issue = await this.issueStore.confirmCreation(scope, async () => {
        this.assertCurrent(ws, resource, token);
        const answer = await this.issueStore.draftApi.publish(id);
        this.assertCurrent(ws, resource, token);
        if (answer.workspace_id !== resource.scope.workspaceID) throw new SessionChangedError();
        this.remove(resource, token, id);
        return answer;
      });
      this.assertCurrent(ws, resource, token);
      await this.refreshTotal(ws, resource, token);
      return issue;
    });
  // Drafts do not participate in ordinary issue layouts; retain only their common read-only facade.
  viewFlags: ViewFlags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: false };
  groups: Map<string, Group> | undefined = undefined;
  getIssueIds = (_groupId?: string, _subGroupId?: string) => undefined;
  getPaginationData = (_groupId?: string, _subGroupId?: string) => undefined;
  getIssueLoader = (_groupId?: string, _subGroupId?: string): TLoader => "loaded";
  getGroupIssueCount = (_groupId?: string, _subGroupId?: string, _isSubGroupCumulative?: boolean) => undefined;
}
