/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueUpdate } from "@nerve/api-client";
import { action, makeObservable, observable } from "mobx";
import { computedFn } from "mobx-utils";
import type { TSubIssuesStateDistribution } from "@nerve/types";
import type { ConfirmedIssueChange } from "@/lib/paged-groups";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueReadToken, IssueWriteScope } from "../root.store";
import { IssueResources } from "./resources";
import type { DetailResource } from "./resource";
import type { IIssueDetail } from "./root.store";
import { WorkItemSubIssueFiltersStore } from "./sub_issues_filter.store";

type HelperKey = "issue_visibility";
type Helpers = Record<HelperKey, string[]>;
export type IIssueSubIssuesStore = IssueSubIssuesStore;
export type IIssueSubIssuesStoreActions = Pick<
  IssueSubIssuesStore,
  "fetchSubIssues" | "createSubIssues" | "updateSubIssue" | "removeSubIssue" | "deleteSubIssue"
>;
export class IssueSubIssuesStore {
  private readonly resources: IssueResources<string[]>;
  readonly filters: WorkItemSubIssueFiltersStore;
  subIssueHelpers: Record<string, Helpers> = {};
  constructor(readonly rootIssueDetailStore: IIssueDetail) {
    this.resources = new IssueResources(this.root);
    this.filters = new WorkItemSubIssueFiltersStore(this);
    makeObservable(this, { subIssueHelpers: observable, setSubIssueHelpers: action });
  }
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }
  subIssuesByIssueId = computedFn((id: string) =>
    this.resources.get(id)?.value?.filter((childID) => this.root.issues.getIssueById(childID)?.parent_id === id)
  );
  stateDistributionByIssueId = computedFn((id: string): TSubIssuesStateDistribution | undefined => {
    const children = this.subIssuesByIssueId(id);
    if (!children) return undefined;
    const distribution: TSubIssuesStateDistribution = {
      backlog: [],
      unstarted: [],
      started: [],
      completed: [],
      cancelled: [],
    };
    for (const childID of children) {
      const child = this.root.issues.getIssueById(childID);
      const group = child && this.root.rootStore.state.getStateById(child.state_id)?.group;
      if (group) distribution[group].push(childID);
    }
    return distribution;
  });
  subIssueHelpersByIssueId = (id: string): Helpers => this.subIssueHelpers[id] ?? { issue_visibility: [] };
  setSubIssueHelpers = (id: string, key: HelperKey, value: string) => {
    const held = this.subIssueHelpersByIssueId(id);
    this.subIssueHelpers[id] = {
      ...held,
      [key]: held[key].includes(value) ? held[key].filter((item) => item !== value) : [...held[key], value],
    };
  };
  getLoading = (id: string) => this.resources.get(id)?.loading ?? false;
  getError = (id: string) => this.resources.get(id)?.error;
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchSubIssues(ws, project, id);
  };
  private current(token: IssueReadToken, scope: IssueWriteScope): void {
    if (!this.root.isReadCurrent(token, [scope.projectID], scope.workspaceID)) throw new SessionChangedError();
    this.root.assertProjectReadable(scope.workspaceSlug, scope.projectID);
  }
  private accept(children: Issue[], parentID: string, scope: IssueWriteScope, token: IssueReadToken): void {
    this.current(token, scope);
    if (
      children.some(
        (child) =>
          child.project_id !== scope.projectID ||
          child.workspace_id !== scope.workspaceID ||
          child.parent_id !== parentID
      )
    )
      throw new SessionChangedError();
    this.root.acceptListIssues(children, token);
  }
  private async states(project: string): Promise<void> {
    const states = this.root.rootStore.state;
    if (states.getProjectStates(project) === undefined) await states.fetchProjectStates(project);
  }
  fetchSubIssues = (ws: string, project: string, parentID: string): Promise<string[]> =>
    this.readChildren(ws, project, parentID, true);
  private async readChildren(ws: string, project: string, parentID: string, verifyParent: boolean): Promise<string[]> {
    const resource = this.resources.entry(ws, project, parentID);
    const scope = this.root.scope(ws, project);
    const value = await resource.fetch(async (read) => {
      const answer = await this.root.detailApi.listSubIssues(parentID);
      if (!read.isCurrent()) throw new SessionChangedError();
      this.accept(answer.data, parentID, scope, read.token);
      await this.states(project);
      if (!read.isCurrent()) throw new SessionChangedError();
      if (verifyParent) await this.root.issues.refreshAuthoritative(parentID);
      if (!read.isCurrent()) throw new SessionChangedError();
      return answer.data.map((child) => child.id);
    });
    if (!value) throw new SessionChangedError();
    return value;
  }
  private refresh(
    resource: DetailResource<string[]>,
    ids: string[],
    scope: IssueWriteScope,
    token: IssueReadToken,
    reread?: () => Promise<unknown>
  ) {
    return resource.refreshAfterConfirmation(async () => {
      this.current(token, scope);
      // Start all endpoint reads so every affected list is marked stale even if one read fails.
      const results = await Promise.allSettled(
        [...new Set(ids)].map((id) => this.root.issues.refreshAuthoritative(id, { afterWrite: true }))
      );
      this.current(token, scope);
      const failed = results.find((result) => result.status === "rejected");
      if (failed?.status === "rejected") throw failed.reason;
      await this.states(scope.projectID);
      if (reread) await reread();
    }, token);
  }
  createSubIssues = (ws: string, project: string, parentID: string, ids: string[]): Promise<void> =>
    this.root.enqueueIssueChanges(
      [parentID, ...ids],
      async () => {
        const resource = this.resources.entry(ws, project, parentID);
        const scope = this.root.scope(ws, project);
        const token = this.root.captureRead();
        const oldParents = ids.flatMap((id) => {
          const old = this.root.issues.getIssueById(id)?.parent_id;
          return old ? [old] : [];
        });
        const answer = await this.root.detailApi.addSubIssues(parentID, { sub_issue_ids: ids });
        this.accept(answer.data, parentID, scope, token);
        // A batch answer does not contain the previous parent/version. Do not guess its list projection.
        this.root.markScopeListsStale(scope);
        const confirmedIDs = answer.data.map((child) => child.id);
        resource.receive(confirmedIDs, token, (held, incoming) => [...new Set([...(held ?? []), ...incoming])]);
        for (const old of oldParents)
          if (old !== parentID)
            this.resources.get(old)?.confirm((held) => held.filter((id) => !confirmedIDs.includes(id)), token);
        await this.refresh(resource, [parentID, ...oldParents], scope, token);
        await Promise.all(
          [...new Set([parentID, ...oldParents, ...ids])].map((id) =>
            this.rootIssueDetailStore.refreshAfterChange(ws, project, id)
          )
        );
      },
      { workspaceSlug: ws, projectID: project }
    );
  /** The root calls this once for its confirmed core change; never sends a second PATCH. */
  confirmCore = async (change: ConfirmedIssueChange, scope: IssueWriteScope): Promise<void> => {
    const before = change.beforeSnapshot.kind === "known" ? change.beforeSnapshot.issue.parent_id : null;
    const after = change.after?.parent_id ?? null;
    if (before === after) return;
    const reloadChildren = !change.after && before && this.resources.get(before);
    const token = this.root.captureRead();
    if (before) this.resources.get(before)?.confirm((held) => held.filter((id) => id !== change.id), token);
    if (after)
      this.resources
        .entry(scope.workspaceSlug, scope.projectID, after)
        .receive([change.id], token, (held, added) => [...new Set([...(held ?? []), ...added])]);
    const parents = [...new Set([before, after].filter((id): id is string => !!id))];
    const resource = this.resources.entry(scope.workspaceSlug, scope.projectID, after ?? before!);
    await this.refresh(
      resource,
      parents,
      scope,
      token,
      reloadChildren ? () => this.readChildren(scope.workspaceSlug, scope.projectID, before!, false) : undefined
    );
  };
  updateSubIssue = async (
    ws: string,
    project: string,
    _parentID: string,
    id: string,
    body: IssueUpdate
  ): Promise<void> => {
    await this.root.projectIssues.updateIssue(ws, project, id, body);
  };
  removeSubIssue = async (ws: string, project: string, parent: string, id: string) =>
    this.updateSubIssue(ws, project, parent, id, { parent_id: null });
  deleteSubIssue = async (ws: string, project: string, _parent: string, id: string) =>
    this.root.projectIssues.removeIssue(ws, project, id);
}
