/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import { autorun, makeObservable, observable } from "mobx";
// types
import type { ApiClient, MemberUser, State, Label } from "@nerve/api-client";
// store
import type { IIssueDetail } from "@/store/issue/issue-details/root.store";
import { IssueDetail } from "@/store/issue/issue-details/root.store";
import type { RootStore } from "@/store/root.store";
// issues data store
import type { IArchivedIssuesFilter, IArchivedIssues } from "./archived";
import { ArchivedIssuesFilter, ArchivedIssues } from "./archived";
import type { IIssueStore } from "./issue.store";
import { IssueStore } from "./issue.store";
import type { ICalendarStore } from "./issue_calendar_view.store";
import { CalendarStore } from "./issue_calendar_view.store";
import type { IIssueKanBanViewStore } from "./issue_kanban_view.store";
import { IssueKanBanViewStore } from "./issue_kanban_view.store";
import type { IProfileIssuesFilter, IProfileIssues } from "./profile";
import { ProfileIssuesFilter, ProfileIssues } from "./profile";
import type { IProjectIssuesFilter, IProjectIssues } from "./project";
import { ProjectIssuesFilter, ProjectIssues } from "./project";
import type { IWorkspaceDraftIssues, IWorkspaceDraftIssuesFilter } from "./workspace-draft";
import { WorkspaceDraftIssues, WorkspaceDraftIssuesFilter } from "./workspace-draft";

export interface IIssueRootStore {
  currentUserId: string | undefined;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  userId: string | undefined; // user profile detail Id
  stateMap: Record<string, State> | undefined;
  labelMap: Record<string, Label> | undefined;
  memberMap: Record<string, MemberUser> | undefined;

  rootStore: RootStore;

  issues: IIssueStore;

  issueDetail: IIssueDetail;

  workspaceDraftIssuesFilter: IWorkspaceDraftIssuesFilter;
  workspaceDraftIssues: IWorkspaceDraftIssues;

  profileIssuesFilter: IProfileIssuesFilter;
  profileIssues: IProfileIssues;

  projectIssuesFilter: IProjectIssuesFilter;
  projectIssues: IProjectIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;
}

export class IssueRootStore implements IIssueRootStore {
  currentUserId: string | undefined = undefined;
  workspaceSlug: string | undefined = undefined;
  projectId: string | undefined = undefined;
  userId: string | undefined = undefined;
  stateMap: Record<string, State> | undefined = undefined;
  labelMap: Record<string, Label> | undefined = undefined;
  memberMap: Record<string, MemberUser> | undefined = undefined;

  rootStore: RootStore;

  issues: IIssueStore;

  issueDetail: IIssueDetail;

  workspaceDraftIssuesFilter: IWorkspaceDraftIssuesFilter;
  workspaceDraftIssues: IWorkspaceDraftIssues;

  profileIssuesFilter: IProfileIssuesFilter;
  profileIssues: IProfileIssues;

  projectIssuesFilter: IProjectIssuesFilter;
  projectIssues: IProjectIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;

  constructor(rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      workspaceSlug: observable.ref,
      projectId: observable.ref,
      userId: observable.ref,
      stateMap: observable,
      labelMap: observable,
      memberMap: observable,
    });

    this.rootStore = rootStore;

    autorun(() => {
      if (rootStore?.user?.data?.id) this.currentUserId = rootStore?.user?.data?.id;
      if (this.workspaceSlug !== rootStore.router.workspaceSlug) this.workspaceSlug = rootStore.router.workspaceSlug;
      if (this.projectId !== rootStore.router.projectId) this.projectId = rootStore.router.projectId;
      if (this.userId !== rootStore.router.userId) this.userId = rootStore.router.userId;
      // the state and label stores' records as they give them, none of a project they no longer give (M3/P8b)
      this.stateMap = rootStore?.state?.stateMap;
      this.labelMap = rootStore?.label?.labelMap;
      if (!isEmpty(rootStore?.memberRoot?.memberMap)) this.memberMap = rootStore?.memberRoot?.memberMap || undefined;
    });

    this.issues = new IssueStore();

    this.issueDetail = new IssueDetail(this);

    this.profileIssuesFilter = new ProfileIssuesFilter(this);
    this.profileIssues = new ProfileIssues(this, this.profileIssuesFilter, api);

    this.workspaceDraftIssuesFilter = new WorkspaceDraftIssuesFilter(this);
    this.workspaceDraftIssues = new WorkspaceDraftIssues(this);

    this.projectIssuesFilter = new ProjectIssuesFilter(this);
    this.projectIssues = new ProjectIssues(this, this.projectIssuesFilter);

    this.archivedIssuesFilter = new ArchivedIssuesFilter(this);
    this.archivedIssues = new ArchivedIssues(this, this.archivedIssuesFilter);

    this.issueKanBanView = new IssueKanBanViewStore(this);
    this.issueCalendarView = new CalendarStore(this);
  }
}
