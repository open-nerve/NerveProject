/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { makeObservable, observable, runInAction } from "mobx";
import type { TLoader } from "@nerve/types";
import { EUserPermissions } from "@nerve/constants";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { Reconciled } from "@/lib/reconciled";
import {
  confirmGroups,
  decodePage,
  mergePages,
  pagePartKey,
  type ConfirmedIssueChange,
  type ListContext,
  type PageAnswer,
  type PagePart,
  type PagedGroups,
  type Group,
  type Boundary,
} from "@/lib/paged-groups";
import { replayGroups } from "@/lib/paged-groups.replay";
import { issueParts } from "@/lib/issue-membership";
import type { VerificationRound } from "../issue-verification";
import type { IIssueRootStore, IssueWriteScope } from "../root.store";

export type IssueListScope =
  | {
      scope: "project";
      workspaceSlug: string;
      workspaceID: string;
      projectID: string;
      query: import("@/services/issue/issue-list.service").ProjectIssueQuery;
    }
  | {
      scope: "member";
      workspaceSlug: string;
      workspaceID: string;
      userID: string;
      query: import("@/services/issue/issue-list.service").MemberIssueQuery;
    };

type ReadRequest = Readonly<{ part: PagePart | null; first: boolean; cursor: string | null; lower: Boundary | null }>;

/** One immutable scope/query, retained when another list is selected. */
export class IssueListBinding {
  readonly reconciled = new Reconciled<PagedGroups>();
  readonly loaders = observable.map<string, TLoader>();
  private readonly loading = new Map<string, symbol>();
  private permissionStale = false;
  private source = 0;
  private visibleParts: readonly PagePart[] = [];
  get isVisible(): boolean {
    return this.visibleParts.length > 0;
  }
  get verifying(): boolean {
    return !this.permissionStale && this.root.verification.isVerifying(this);
  }
  setVisibleParts(parts: readonly PagePart[]): void {
    this.visibleParts = parts;
  }
  private readonly failedReads = new Map<string, ReadRequest>();
  error: unknown;
  constructor(
    readonly root: IIssueRootStore,
    readonly scope: IssueListScope
  ) {
    makeObservable<this, "permissionStale">(this, { error: observable.ref, permissionStale: observable });
    for (const value of Object.values(scope.query)) if (Array.isArray(value)) Object.freeze(value);
    Object.freeze(scope.query);
    Object.freeze(scope);
    root.registerList(this);
  }
  get context(): ListContext {
    return {
      ...this.scope,
      currentUserID: this.root.currentUserId ?? "",
      pendingIDs: this.root.pendingIDs,
      catalog: {
        states: new Map(Object.entries(this.root.stateMap ?? {})),
        labels: new Map(Object.entries(this.root.labelMap ?? {})),
        members: new Map(Object.entries(this.root.memberMap ?? {})),
      },
    };
  }
  covers(scope: IssueWriteScope): boolean {
    return (
      this.scope.workspaceID === scope.workspaceID &&
      (this.scope.scope === "project"
        ? this.scope.projectID === scope.projectID
        : !this.scope.query.project_id?.length || this.scope.query.project_id.includes(scope.projectID))
    );
  }
  invalidatePermission(): void {
    this.reconciled.invalidate();
    this.failedReads.clear();
    this.loading.clear();
    runInAction(() => {
      this.permissionStale = true;
      this.error = undefined;
      this.loaders.clear();
    });
  }
  get value(): PagedGroups | undefined {
    return this.permissionStale ? undefined : this.reconciled.value;
  }
  private scopeRole(): number | undefined {
    const stores = this.root.rootStore;
    if (stores.workspaceRoot.getWorkspaceBySlug(this.scope.workspaceSlug)?.id !== this.scope.workspaceID)
      return undefined;
    const permission = stores.user.permission;
    return this.scope.scope === "project"
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(this.scope.workspaceSlug, this.scope.projectID)
      : permission.getWorkspaceRoleByWorkspaceSlug(this.scope.workspaceSlug);
  }
  private allowsRole(role: number | undefined): boolean {
    return this.scope.scope === "member" || this.scope.query.archived === "true"
      ? role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER
      : role !== undefined;
  }
  private async read(request: ReadRequest): Promise<PageAnswer> {
    const role = this.scopeRole();
    if (!this.allowsRole(role)) throw new SessionChangedError();
    const currentScope = () => this.scopeRole() === role && this.allowsRole(role);
    const { part, first, cursor, lower } = request;
    const token = this.root.captureRead(this.scope.scope === "member");
    const query = {
      ...this.scope.query,
      ...(part
        ? {
            ...(this.scope.query.group_by ? { group_key: part.group } : {}),
            ...(part.subgroup === undefined ? {} : { sub_group_key: part.subgroup }),
            ...(!first && cursor ? { cursor } : {}),
          }
        : {}),
    };
    const page =
      this.scope.scope === "project"
        ? await this.root.listApi.listProject(this.scope.projectID, query)
        : await this.root.listApi.listMember(this.scope.workspaceSlug, this.scope.userID, {
            ...query,
            relation: this.scope.query.relation,
          });
    const projects = this.scope.scope === "project" ? [this.scope.projectID] : (this.scope.query.project_id ?? []);
    if (!currentScope() || !this.root.isReadCurrent(token, projects, this.scope.workspaceID))
      throw new SessionChangedError();
    const issues = page.groups
      ? page.groups.flatMap((group) =>
          group.subgroups ? group.subgroups.flatMap((cell) => cell.data) : (group.data ?? [])
        )
      : (page.data ?? []);
    if (
      issues.some(
        (issue) =>
          issue.workspace_id !== this.scope.workspaceID ||
          (this.scope.scope === "project" && issue.project_id !== this.scope.projectID) ||
          (this.scope.scope === "member" &&
            !!this.scope.query.project_id?.length &&
            !this.scope.query.project_id.includes(issue.project_id))
      )
    )
      throw new Error("Issue page does not match its scope");
    const returnedProjects = [...new Set(issues.map((issue) => issue.project_id))];
    const returnedCurrent = () =>
      !returnedProjects.length || this.root.isReadCurrent(token, returnedProjects, this.scope.workspaceID);
    if (!returnedCurrent()) throw new SessionChangedError();
    const projectStore = this.root.rootStore.projectRoot.project;
    await Promise.all(
      returnedProjects.map(async (projectID) => {
        if (!projectStore.getProjectById(projectID)) await projectStore.fetchProject(projectID);
      })
    );
    if (!returnedCurrent() || !currentScope() || !this.root.isReadCurrent(token, projects, this.scope.workspaceID))
      throw new SessionChangedError();
    if (issues.some((issue) => !this.root.canSeeIssue(issue)))
      throw new Error("Required issue project permission is unavailable");
    await this.prepareCatalog(issues);
    if (!returnedCurrent() || !currentScope() || !this.root.isReadCurrent(token, projects, this.scope.workspaceID))
      throw new SessionChangedError();
    const known = this.reconciled.value;
    const rowParts: PagePart[] = part
      ? []
      : [...(known?.groups ?? [])].flatMap(([group, data]) =>
          data.subgroups ? [...data.subgroups.keys()].map((subgroup) => ({ group, subgroup })) : [{ group }]
        );
    let answer = decodePage(
      page,
      {
        sourceID: ++this.source,
        part: this.scope.query.group_by ? part : null,
        first,
        rowParts,
        resetCounters: !part
          ? [...(known?.subgroupTotals?.keys() ?? [])].map((subgroup) => ({ kind: "subgroup", subgroup }))
          : [],
        ranges: new Map(part ? [[pagePartKey(part), { lower, upper: null, complete: false }]] : []),
      },
      this.context
    );
    // Decode every row before touching canonical; malformed trailing rows cannot partially publish a page.
    if (this.root.verification.blocksPage(this, token)) throw new SessionChangedError();
    this.root.acceptListIssues(issues, token);
    for (const issue of issues) {
      if (this.root.issues.held(issue.id) === null)
        answer = {
          ...confirmGroups(
            answer,
            { id: issue.id, beforeSnapshot: { kind: "known", issue }, after: null },
            this.context
          ).value,
          read: answer.read,
        };
      else if (this.root.verification.isUncertain(issue.id))
        for (const affected of issueParts(issue, this.context)) {
          const group = answer.groups.get(affected.group);
          const cell = affected.subgroup === undefined ? group : group?.subgroups?.get(affected.subgroup);
          if (cell) cell.stale = true;
        }
    }
    return answer;
  }
  async prepareCatalog(issues: Issue[]): Promise<void> {
    const stores = this.root.rootStore;
    const order = this.scope.query.order_by?.replace(/^-/, "");
    const states =
      order === "state_name" ||
      this.scope.query.group_by === "state_group" ||
      ("sub_group_by" in this.scope.query && this.scope.query.sub_group_by === "state_group") ||
      !!this.scope.query.state_group?.length;
    const labels = order === "label_name";
    const members = order === "assignee_name";
    if (!states && !labels && !members) return;
    if (states || labels)
      await Promise.all(
        [...new Set(issues.map((issue) => issue.project_id))].map(async (project) => {
          this.root.assertActive();
          if (!stores.projectRoot.project.getProjectById(project))
            await stores.projectRoot.project.fetchProject(project);
          this.root.assertActive();
          await Promise.all([
            states &&
            (stores.state.getProjectStates(project) === undefined ||
              issues.some((issue) => issue.project_id === project && !stores.state.getStateById(issue.state_id)))
              ? stores.state.fetchProjectStates(project)
              : undefined,
            labels &&
            (stores.label.getProjectLabels(project) === undefined ||
              issues.some(
                (issue) => issue.project_id === project && issue.label_ids.some((id) => !stores.label.getLabelById(id))
              ))
              ? stores.label.fetchProjectLabels(project)
              : undefined,
          ]);
          this.root.assertActive();
        })
      );
    if (members && issues.some((issue) => issue.assignee_ids.length)) {
      this.root.assertActive();
      if (
        !stores.memberRoot.workspace.getMemberships(this.scope.workspaceSlug) ||
        issues.some((issue) => issue.assignee_ids.some((id) => !stores.memberRoot.getUserDetails(id)))
      )
        await stores.memberRoot.workspace.fetchWorkspaceMembers({
          id: this.scope.workspaceID,
          slug: this.scope.workspaceSlug,
        });
      this.root.assertActive();
    }
    const catalog = this.context.catalog;
    if (
      issues.some(
        (issue) =>
          (states && !catalog.states.has(issue.state_id)) ||
          (labels && issue.label_ids.some((id) => !catalog.labels.has(id))) ||
          (members && issue.assignee_ids.some((id) => !catalog.members.has(id)))
      )
    )
      throw new Error("Required issue directory is unavailable");
  }
  private async fetch(
    part: PagePart | null,
    first: boolean,
    retry?: ReadRequest,
    refreshing?: VerificationRound
  ): Promise<PagedGroups | undefined> {
    this.root.assertActive();
    const projectIDs = this.scope.scope === "project" ? [this.scope.projectID] : (this.scope.query.project_id ?? []);
    const identity = this.root.permissionReads.identity(
      this.scope.workspaceID,
      projectIDs,
      this.scope.scope === "member"
    );
    const round = this.root.verification.begin();
    const heldGroup = part ? this.group(part.group, part.subgroup) : undefined;
    const request =
      retry ??
      structuredClone({
        part,
        first,
        cursor: first ? null : (heldGroup?.nextCursor ?? null),
        lower: first ? null : (heldGroup?.cursorBoundary ?? null),
      });
    const key = part ? pagePartKey(part) : "whole";
    const loading = Symbol(key);
    this.loading.set(key, loading);
    runInAction(() => this.loaders.set(key, first ? "init-loader" : "pagination"));
    try {
      const value = part
        ? await this.reconciled.fetchPart(
            key,
            () => this.read(request),
            (held, answer) => mergePages(held, this.answer(answer), this.context)
          )
        : this.reconciled.value === undefined || this.permissionStale
          ? await this.reconciled.fetch(() => this.read(request))
          : await this.refreshWhole(request);
      if (value === undefined) return undefined;
      if (!refreshing) await this.root.verification.drain(this, round);
      if (
        !this.root.active ||
        identity !==
          this.root.permissionReads.identity(this.scope.workspaceID, projectIDs, this.scope.scope === "member")
      )
        return undefined;
      if (part === null) this.failedReads.clear();
      else this.failedReads.delete(key);
      runInAction(() => {
        if (part === null) this.permissionStale = false;
        if (!this.failedReads.size) this.error = undefined;
      });
      return value;
    } catch (error) {
      if (this.root.active && !(error instanceof SessionChangedError))
        runInAction(() => {
          this.failedReads.set(key, request);
          this.error = error;
        });
      throw error;
    } finally {
      this.root.verification.end(round);
      if (this.root.active && this.loading.get(key) === loading) {
        this.loading.delete(key);
        runInAction(() => this.loaders.delete(key));
      }
    }
  }
  private refreshWhole(request: ReadRequest): Promise<PagedGroups | undefined> {
    this.reconciled.invalidate();
    return this.reconciled.fetchPart(
      "whole",
      () => this.read(request),
      (held, answer) => mergePages(held, this.answer(answer), this.context)
    );
  }
  private answer(value: PagedGroups): PageAnswer {
    if (!value.read) throw new Error("Missing original page metadata");
    return { ...value, read: value.read };
  }
  fetchFirst = () => this.fetch(null, true);
  retryFailed = async (): Promise<void> => {
    this.root.assertActive();
    if (this.permissionStale) {
      await this.fetchFirst();
      return;
    }
    if (await this.root.verification.retry(this)) return;
    const requests = [...this.failedReads.values()];
    if (!requests.length) {
      await this.fetchFirst();
      return;
    }
    await Promise.all(requests.map((request) => this.fetch(request.part, request.first, request)));
  };
  fetchNext = (group = "", subgroup?: string) => {
    if (!this.group(group, subgroup)?.nextCursor) return Promise.resolve(undefined);
    return this.fetch({ group, subgroup }, false);
  };
  refreshStale = async (visible?: readonly PagePart[]): Promise<void> => {
    const value = this.reconciled.value;
    if (!value) return;
    const parts = visible
      ? visible.filter((part) => this.group(part.group, part.subgroup)?.stale)
      : [...value.groups].flatMap(([group, data]) =>
          data.subgroups
            ? [...data.subgroups].filter(([, cell]) => cell.stale).map(([subgroup]) => ({ group, subgroup }))
            : data.stale
              ? [{ group }]
              : []
        );
    await Promise.all(parts.map((part) => this.fetch(part, true)));
  };
  acceptConfirmed(change: ConfirmedIssueChange, heldSources: ReadonlySet<number>): void {
    if (!this.root.active) return;
    this.reconciled.confirm((value, raw) => {
      const result = raw
        ? replayGroups(value, change, this.answer(raw), this.context)
        : confirmGroups(value, change, this.context, heldSources);
      for (const id of result.verifyIds) this.root.verification.require(id, this);
      return result.value;
    });
  }
  refreshVerification(round: VerificationRound): Promise<PagedGroups | undefined> {
    return this.fetch(null, true, undefined, round);
  }
  markVerificationStale(id?: string, error?: unknown): void {
    if (!this.root.active) return;
    if (error)
      runInAction(() => {
        this.error = error;
      });
    const issue = id ? this.root.issues.held(id) : undefined;
    if (issue) this.markMembershipStale(issue, issue);
    else
      this.reconciled.confirm((value) => {
        const next = structuredClone(value);
        for (const group of next.groups.values()) {
          group.stale = true;
          for (const cell of group.subgroups?.values() ?? []) cell.stale = true;
        }
        return next;
      });
  }
  clearVerificationError(error: unknown): void {
    if (this.error !== error) return;
    this.failedReads.clear();
    runInAction(() => {
      this.error = undefined;
    });
  }
  markMembershipStale(before: Issue | undefined, after: Issue | undefined): void {
    this.reconciled.confirm((value) => {
      const next = structuredClone(value);
      for (const part of issueParts(before, this.context).concat(issueParts(after, this.context))) {
        const group = next.groups.get(part.group);
        const cell = part.subgroup === undefined ? group : group?.subgroups?.get(part.subgroup);
        if (cell) cell.stale = true;
      }
      return next;
    });
  }
  markUnavailable(error: unknown): void {
    this.failedReads.clear();
    this.reconciled.invalidate();
    runInAction(() => {
      this.error = error;
    });
    this.reconciled.confirm((value) => {
      const next = structuredClone(value);
      for (const group of next.groups.values()) {
        group.stale = true;
        for (const cell of group.subgroups?.values() ?? []) cell.stale = true;
      }
      return next;
    });
  }
  group(group = "", subgroup?: string): Group | undefined {
    const outer = this.value?.groups.get(group);
    return subgroup === undefined ? outer : outer?.subgroups?.get(subgroup);
  }
}
