/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { computed, makeObservable, observable, runInAction, toJS } from "mobx";
// types
import type {
  ApiClient,
  MemberUser,
  State,
  Label,
  Issue,
  IssueCreate,
  IssueDetail as IssueDetailAnswer,
  IssueUpdate,
} from "@nerve/api-client";
// store
import type { IIssueDetail } from "@/store/issue/issue-details/root.store";
import { IssueDetail } from "@/store/issue/issue-details/root.store";
import type { RootStore } from "@/store/root.store";
// issues data store
import type { IArchivedIssuesFilter, IArchivedIssues } from "./archived";
import { ArchivedIssuesFilter, ArchivedIssues } from "./archived";
import type { IIssueStore } from "./issue.store";
import { IssueStore } from "./issue.store";
import { IssueVerification } from "./issue-verification";
import type { ICalendarStore } from "./issue_calendar_view.store";
import { CalendarStore } from "./issue_calendar_view.store";
import type { IIssueKanBanViewStore } from "./issue_kanban_view.store";
import { IssueKanBanViewStore } from "./issue_kanban_view.store";
import type { IProfileIssuesFilter, IProfileIssues } from "./profile";
import { ProfileIssuesFilter, ProfileIssues } from "./profile";
import type { IProjectIssuesFilter, IProjectIssues } from "./project";
import { ProjectIssuesFilter, ProjectIssues } from "./project";
import type { IWorkspaceDraftIssues } from "./workspace-draft";
import { WorkspaceDraftIssues } from "./workspace-draft";

import { IssueApiService } from "@/services/issue/issue-api.service";
import { IssueListService } from "@/services/issue/issue-list.service";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
import { compareIssueTime } from "@/lib/issue-order";
import type { Before, ConfirmedIssueChange } from "@/lib/paged-groups";
import { heldCounterSources } from "@/lib/paged-groups.evidence";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import { PermissionReads } from "./permission-reads";

export type IIssueRootStore = IssueRootStore;
export type IssueWriteScope = Readonly<{ workspaceSlug: string; projectID: string; workspaceID: string }>;
export type IssueReadToken = Readonly<{ owner: symbol; serial: number; permission: number; member: boolean }>;
type PendingWrite = {
  before: Before;
  sources: Map<IssueListBinding, ReadonlySet<number>>;
  observed: { version: string; fields: string }[];
};

export class IssueRootStore {
  get currentUserId(): string | undefined {
    return this.rootStore.user?.data?.id;
  }
  get workspaceSlug(): string | undefined {
    return this.rootStore.router.workspaceSlug;
  }
  get projectId(): string | undefined {
    return this.rootStore.router.projectId;
  }
  get userId(): string | undefined {
    return this.rootStore.router.userId;
  }
  get stateMap(): Record<string, State> | undefined {
    return this.rootStore.state?.stateMap;
  }
  get labelMap(): Record<string, Label> | undefined {
    return this.rootStore.label?.labelMap;
  }
  get memberMap(): Record<string, MemberUser> | undefined {
    return this.rootStore.memberRoot?.memberMap;
  }

  rootStore: RootStore;

  issues: IIssueStore;

  issueDetail: IIssueDetail;

  workspaceDraftIssues: IWorkspaceDraftIssues;

  profileIssuesFilter: IProfileIssuesFilter;
  profileIssues: IProfileIssues;

  projectIssuesFilter: IProjectIssuesFilter;
  projectIssues: IProjectIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;

  readonly owner = Symbol("issue-root");
  active = true;
  readonly api: IssueApiService;
  readonly listApi: IssueListService;
  private serial = 0;
  get readSerial(): number {
    return this.serial;
  }
  readonly verification = new IssueVerification(this);
  private readonly deletions = new Map<string, { serial: number; workspaceID: string }>();
  private readonly lists = new Set<IssueListBinding>();
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();
  private readonly pending = new Map<string, PendingWrite>();
  readonly permissionReads: PermissionReads;
  get pendingIDs(): ReadonlySet<string> {
    return new Set(this.pending.keys());
  }

  constructor(rootStore: RootStore, api: ApiClient) {
    this.api = new IssueApiService(api);
    this.listApi = new IssueListService(api);
    makeObservable(this, {
      active: observable,
      currentUserId: computed,
      workspaceSlug: computed,
      projectId: computed,
      userId: computed,
      stateMap: computed,
      labelMap: computed,
      memberMap: computed,
    });

    this.rootStore = rootStore;
    this.permissionReads = new PermissionReads(rootStore, (workspaceID, projectID, memberOnly) => {
      for (const list of this.lists)
        if (
          list.scope.workspaceID === workspaceID &&
          (!memberOnly || list.scope.scope === "member") &&
          (!projectID || list.covers({ workspaceID, projectID, workspaceSlug: list.scope.workspaceSlug }))
        )
          list.invalidatePermission();
    });

    this.issues = new IssueStore(this);

    this.issueDetail = new IssueDetail(this);

    this.profileIssuesFilter = new ProfileIssuesFilter(this);
    this.profileIssues = new ProfileIssues(this, this.profileIssuesFilter);

    this.workspaceDraftIssues = new WorkspaceDraftIssues(this);

    this.projectIssuesFilter = new ProjectIssuesFilter(this, api);
    this.projectIssues = new ProjectIssues(this, this.projectIssuesFilter);

    this.archivedIssuesFilter = new ArchivedIssuesFilter(this);
    this.archivedIssues = new ArchivedIssues(this, this.archivedIssuesFilter);

    this.issueKanBanView = new IssueKanBanViewStore(this);
    this.issueCalendarView = new CalendarStore(this);
  }
  assertActive(): void {
    if (!this.active) throw new SessionChangedError();
  }
  retire(): void {
    if (!this.active) return;
    this.permissionReads.retire();
    runInAction(() => {
      this.active = false;
    });
    for (const list of this.lists) list.reconciled.invalidate();
  }
  registerList(list: IssueListBinding): void {
    this.assertActive();
    this.lists.add(list);
  }
  captureRead(member = false): IssueReadToken {
    this.assertActive();
    return Object.freeze({
      owner: this.owner,
      serial: ++this.serial,
      permission: this.permissionReads.capture(),
      member,
    });
  }
  isReadCurrent(token: IssueReadToken, projectIDs: string[], workspaceID?: string): boolean {
    return (
      this.active &&
      token.owner === this.owner &&
      this.permissionReads.current(token.permission, workspaceID, projectIDs, token.member) &&
      (projectIDs.length
        ? projectIDs.every((id) => token.serial >= (this.deletions.get(id)?.serial ?? 0))
        : [...this.deletions.values()].every(
            (deletion) =>
              (workspaceID !== undefined && deletion.workspaceID !== workspaceID) || token.serial >= deletion.serial
          ))
    );
  }
  scope(workspaceSlug: string, projectID: string): IssueWriteScope {
    const workspaceID =
      this.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id ??
      this.rootStore.projectRoot.project.getProjectById(projectID)?.workspace_id;
    if (!workspaceID) throw new Error("Issue workspace is not loaded");
    return { workspaceSlug, projectID, workspaceID };
  }
  assertProjectReadable(workspaceSlug: string, projectID: string): void {
    this.assertActive();
    if (
      this.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectID) === undefined
    )
      throw new SessionChangedError();
  }
  canSeeIssue(issue: Issue): boolean {
    if (!this.active) return false;
    const workspace = this.rootStore.workspaceRoot.workspaces?.find((item) => item.id === issue.workspace_id);
    return (
      !!workspace &&
      this.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, issue.project_id) !==
        undefined
    );
  }
  acceptListIssues(issues: Issue[], token: IssueReadToken): void {
    for (const issue of issues) this.issues.acceptRead(issue, token);
  }
  listsForIssue(id: string): IssueListBinding[] {
    const issue = this.issues.held(id);
    return issue
      ? [...this.lists].filter((list) =>
          list.covers({
            projectID: issue.project_id,
            workspaceID: issue.workspace_id,
            workspaceSlug: list.scope.workspaceSlug,
          })
        )
      : [];
  }
  observeRead(before: Issue | undefined, after: Issue): void {
    const pending = this.pending.get(after.id);
    pending?.observed.push({ version: after.updated_at, fields: this.fields(after) });
    if (!pending && before && this.fields(before) !== this.fields(after))
      for (const list of this.lists) list.markMembershipStale(before, after);
  }
  invalidateProjectReads(projectID: string, workspaceID: string): void {
    this.deletions.set(projectID, { serial: ++this.serial, workspaceID });
    for (const list of this.lists)
      if (list.covers({ projectID, workspaceID, workspaceSlug: list.scope.workspaceSlug })) {
        list.reconciled.invalidate();
        list.markVerificationStale();
      }
  }
  private fields(issue: Issue): string {
    return JSON.stringify([
      issue.project_id,
      issue.workspace_id,
      issue.name,
      issue.sort_order,
      issue.state_id,
      issue.priority,
      issue.label_ids,
      issue.assignee_ids,
      issue.parent_id,
      issue.created_by_id,
      issue.start_date,
      issue.target_date,
      issue.archived_at,
      issue.sub_issues_count,
      issue.link_count,
    ]);
  }
  private queue(id: string) {
    let queue = this.queues.get(id);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(id, queue);
    }
    return queue;
  }
  private async write(
    id: string | undefined,
    scope: IssueWriteScope,
    request: (held?: Issue) => Promise<IssueDetailAnswer | null>,
    drop?: ConfirmedIssueChange["drop"]
  ): Promise<IssueDetailAnswer | null> {
    this.assertActive();
    let initial = id ? this.issues.getIssueById(id) : undefined;
    while (initial) {
      for (const list of this.lists)
        if (list.covers(scope) && list.reconciled.value !== undefined) {
          // Each directory wait may change canonical; freeze only after this serial readiness pass.
          // oxlint-disable-next-line no-await-in-loop
          await list.prepareCatalog([initial]);
        }
      this.assertActive();
      const latest = id ? this.issues.getIssueById(id) : undefined;
      if (latest === initial) break;
      initial = latest;
    }
    this.assertActive();
    const held = id ? this.issues.getIssueById(id) : undefined;
    const pending: PendingWrite = {
      before: id
        ? held
          ? { kind: "known", issue: structuredClone(toJS(held)) }
          : { kind: "unknown" }
        : { kind: "absent" },
      sources: new Map([...this.lists].map((list) => [list, heldCounterSources(list.reconciled.value)])),
      observed: [],
    };
    if (id) this.pending.set(id, pending);
    try {
      const permission = this.permissionReads.capture();
      const after = await request(held);
      this.assertActive();
      const prepared = new Set<IssueListBinding>();
      const unavailable = new Set<IssueListBinding>();
      // A binding can be created while a request or directory is out. Each owns its query and proof.
      for (const list of this.lists)
        if (list.covers(scope) && !prepared.has(list)) {
          prepared.add(list);
          try {
            await list.prepareCatalog([
              ...(pending.before.kind === "known" ? [pending.before.issue] : []),
              ...(after ? [after] : []),
            ]);
          } catch (error) {
            this.assertActive();
            if (error instanceof SessionChangedError) throw error;
            // The issue write already succeeded. Keep that fact; only this list needs authoritative replacement.
            unavailable.add(list);
            list.markUnavailable(error);
          }
        }
      this.assertActive();
      let before = pending.before;
      if (after && before.kind === "known") {
        const old = before.issue;
        if (
          pending.observed.some(
            (read) =>
              compareIssueTime(read.version, after.updated_at) > 0 ||
              (compareIssueTime(read.version, old.updated_at) > 0 &&
                (compareIssueTime(read.version, after.updated_at) === 0
                  ? read.fields !== this.fields(after)
                  : read.fields !== this.fields(old)))
          )
        )
          before = { kind: "unknown" };
      }
      const change: ConfirmedIssueChange = { id: after?.id ?? id!, beforeSnapshot: before, after, drop };
      if (!after) {
        this.invalidateProjectReads(scope.projectID, scope.workspaceID);
        for (const child of this.issues.knownDescendants(change.id)) {
          const deletion = { id: child.id, beforeSnapshot: { kind: "known" as const, issue: child }, after: null };
          for (const list of this.lists)
            if (list.covers(scope)) list.acceptConfirmed(deletion, pending.sources.get(list) ?? new Set());
          this.issues.confirmChange(deletion);
        }
      }
      for (const list of this.lists)
        if (list.covers(scope) && !unavailable.has(list))
          list.acceptConfirmed(change, pending.sources.get(list) ?? new Set());
      this.issues.confirmChange(change, permission);
      return after;
    } catch (error) {
      this.assertActive();
      throw error;
    } finally {
      if (id && this.pending.get(id) === pending) this.pending.delete(id);
    }
  }
  createIssue(scope: IssueWriteScope, input: IssueCreate): Promise<IssueDetailAnswer> {
    const body: IssueCreate = {
      name: input.name,
      description_html: input.description_html,
      state_id: input.state_id,
      priority: input.priority,
      assignee_ids: input.assignee_ids,
      label_ids: input.label_ids,
      start_date: input.start_date,
      target_date: input.target_date,
      parent_id: input.parent_id,
    };
    return this.write(undefined, scope, () => this.api.create(scope.projectID, body)).then((answer) => answer!);
  }
  updateIssue(
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop?: ConfirmedIssueChange["drop"],
    operationScope?: IssueWriteScope
  ): Promise<IssueDetailAnswer> {
    return this.queue(id)(async () => {
      this.assertActive();
      const snapshot = this.issues.getIssueById(id);
      if (!snapshot) throw new Error("Issue has not been read");
      const scope = operationScope ?? this.scope(this.workspaceSlug!, snapshot.project_id);
      return (await this.write(
        id,
        scope,
        (held) => {
          if (!held) throw new Error("Issue has not been read");
          const input = makeBody(held);
          const body: IssueUpdate = {
            name: input.name,
            description_html: input.description_html,
            state_id: input.state_id,
            priority: input.priority,
            assignee_ids: input.assignee_ids,
            label_ids: input.label_ids,
            sort_order: input.sort_order,
            start_date: input.start_date,
            target_date: input.target_date,
            parent_id: input.parent_id,
            skip_activity: input.skip_activity,
          };
          if (
            body.skip_activity !== undefined &&
            (typeof body.skip_activity !== "boolean" ||
              body.description_html === undefined ||
              Object.entries(body).some(
                ([key, value]) => value !== undefined && !["description_html", "skip_activity"].includes(key)
              ))
          )
            throw new Error("skip_activity requires description_html alone");
          return this.api.update(id, body);
        },
        drop
      ))!;
    });
  }
  deleteIssue(id: string, scope: IssueWriteScope): Promise<void> {
    return this.queue(id)(async () => {
      await this.write(id, scope, async () => {
        await this.api.delete(id);
        return null;
      });
    });
  }
  archiveIssue(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.queue(id)(async () => (await this.write(id, scope, () => this.api.archive(id)))!);
  }
  unarchiveIssue(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.queue(id)(async () => (await this.write(id, scope, () => this.api.unarchive(id)))!);
  }
}
