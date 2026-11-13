/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { computed, makeObservable, observable, runInAction } from "mobx";
// types
import type { ApiClient, MemberUser, State, Label, Issue } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
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
import { CycleIssues, type ICycleIssues } from "./cycle/issue.store";
import { CyclePriorityIssues } from "./cycle/priority-issues.store";
import { PagePreferencesFilter, type IPagePreferencesFilter } from "./helpers/page-preferences-filter.store";
import type { IWorkspaceDraftIssues } from "./workspace-draft";
import { WorkspaceDraftIssues } from "./workspace-draft";
import { IssueWrites } from "./issue-writes";
import { membershipFields } from "./issue-write-fields";

import { IssueApiService } from "@/services/issue/issue-api.service";
import { IssueListService } from "@/services/issue/issue-list.service";
import { IssueSearchService } from "@/services/issue/issue-search.service";
import { CyclePreferencesService } from "@/services/planning/cycle-preferences.service";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import { PermissionReads } from "./permission-reads";
import type { DetailRegistration } from "./issue-details/resource";
import { IssueDetailService } from "@/services/issue/issue-detail.service";
import { IssueCommentService } from "@/services/issue/issue_comment.service";
import { WorkspaceDraftService } from "@/services/issue/workspace_draft.service";
import { WorkItemVersionService } from "@/services/issue/work_item_version.service";
import { IssueActivityService } from "@/services/issue/issue_activity.service";
import { IssueReactionService } from "@/services/issue/issue_reaction.service";

export type IIssueRootStore = IssueRootStore;
export type IssueWriteScope = Readonly<{ workspaceSlug: string; projectID: string; workspaceID: string }>;
export type IssueReadToken = Readonly<{ owner: symbol; serial: number; permission: number; member: boolean }>;

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

  /**
   * The cycle page's preferences (`/me/cycles/{id}/preferences`), by cycle id (P5b spec 2.4, P6b spec 2.2): no route
   * followed, the page passes the cycle of its route params on each call; it ends with this generation (assertActive)
   * as the project's.
   */
  cycleIssuesFilter: IPagePreferencesFilter;
  /** The cycle page's list, one binding per cycle, registered in this generation as every list is. */
  cycleIssues: ICycleIssues;
  /**
   * The cycles' priority work items of the card of the cycle in progress: their own list store, never the cycle
   * page's (P5b R7), the one of this generation (re-66 M4).
   */
  cyclePriorityIssues: CyclePriorityIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;

  readonly owner = Symbol("issue-root");
  active = true;
  readonly api: IssueApiService;
  readonly detailApi: IssueDetailService;
  readonly commentApi: IssueCommentService;
  readonly draftApi: WorkspaceDraftService;
  readonly versionApi: WorkItemVersionService;
  readonly activityApi: IssueActivityService;
  readonly reactionApi: IssueReactionService;
  readonly listApi: IssueListService;
  readonly searchApi: IssueSearchService;
  private serial = 0;
  get readSerial(): number {
    return this.serial;
  }
  readonly verification = new IssueVerification(this);
  private readonly deletions = new Map<string, { serial: number; workspaceID: string }>();
  private readonly lists = new Set<IssueListBinding>();
  private readonly resources = new Set<DetailRegistration>();
  readonly permissionReads: PermissionReads;
  /** This generation's writes: the queues, the writes in flight, the confirmation and the commands (P6b R1). */
  readonly writes = new IssueWrites(this);
  /** The lists registered in this generation, read by the writes; never a second copy. */
  get registeredLists(): ReadonlySet<IssueListBinding> {
    return this.lists;
  }

  constructor(rootStore: RootStore, api: ApiClient) {
    this.api = new IssueApiService(api);
    this.detailApi = new IssueDetailService(api);
    this.commentApi = new IssueCommentService(api);
    this.draftApi = new WorkspaceDraftService(api);
    this.versionApi = new WorkItemVersionService(api);
    this.activityApi = new IssueActivityService(api);
    this.reactionApi = new IssueReactionService(api);
    this.listApi = new IssueListService(api);
    this.searchApi = new IssueSearchService(api);
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
      for (const resource of this.resources)
        if (resource.affects(workspaceID, projectID, memberOnly)) resource.invalidate();
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

    this.cycleIssuesFilter = new PagePreferencesFilter(this, new CyclePreferencesService(api), EIssuesStoreType.CYCLE);
    this.cycleIssues = new CycleIssues(this, this.cycleIssuesFilter);
    this.cyclePriorityIssues = new CyclePriorityIssues(this);

    this.archivedIssuesFilter = new ArchivedIssuesFilter(this);
    this.archivedIssues = new ArchivedIssues(this, this.archivedIssuesFilter);

    this.issueKanBanView = new IssueKanBanViewStore();
    this.issueCalendarView = new CalendarStore(this);
  }
  assertActive(): void {
    if (!this.active) throw new SessionChangedError();
  }
  retire(): void {
    if (!this.active) return;
    this.permissionReads.retire();
    this.issueCalendarView.dispose();
    runInAction(() => {
      this.active = false;
    });
    for (const list of this.lists) list.reconciled.invalidate();
    for (const resource of this.resources) resource.dispose();
  }
  registerList(list: IssueListBinding): void {
    this.assertActive();
    this.lists.add(list);
  }
  registerResource(resource: DetailRegistration): () => void {
    this.assertActive();
    this.resources.add(resource);
    return () => {
      this.resources.delete(resource);
    };
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
  /** A workspace owner's write uses the same permission history, without project or issue deletion watermarks. */
  captureWorkspaceWrite(workspaceID: string): () => boolean {
    const permission = this.permissionReads.capture();
    return () => this.active && this.permissionReads.currentWorkspace(permission, workspaceID);
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
  /** A related command may have changed an unread old parent in any covered group. */
  markScopeListsStale(scope: IssueWriteScope): void {
    for (const list of this.lists) if (list.covers(scope)) list.markVerificationStale();
  }
  observeRead(before: Issue | undefined, after: Issue): void {
    const pending = this.writes.observePendingRead(after);
    if (!pending && before && membershipFields(before) !== membershipFields(after))
      for (const list of this.lists) list.markMembershipStale(before, after);
  }
  invalidateProjectReads(projectID: string, workspaceID: string): void {
    this.deletions.set(projectID, { serial: ++this.serial, workspaceID });
    for (const resource of this.resources) if (resource.affects(workspaceID, projectID)) resource.invalidate();
    for (const list of this.lists)
      if (list.covers({ projectID, workspaceID, workspaceSlug: list.scope.workspaceSlug })) {
        list.reconciled.invalidate();
        list.markVerificationStale();
      }
  }
}
