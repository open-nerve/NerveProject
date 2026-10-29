/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import type { TLoader, TProfileViews } from "@nerve/types";
import { memberListQuery, toMemberFilters, type IssueListOptions } from "@/lib/issue-filters";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IIssueRootStore } from "../root.store";
import type { IProfileIssuesFilter } from "./filter.store";
export type IProfileIssues = ProfileIssues;
export class ProfileIssues extends BaseIssuesStore {
  currentView: TProfileViews = "assigned";
  constructor(
    root: IIssueRootStore,
    readonly issueFilterStore: IProfileIssuesFilter
  ) {
    super(root, issueFilterStore);
    makeObservable(this, { currentView: observable });
  }
  get viewFlags() {
    return { enableQuickAdd: false, enableIssueCreation: this.currentView !== "subscribed", enableInlineEditing: true };
  }
  setViewId = (view: TProfileViews) => {
    runInAction(() => {
      this.currentView = view;
    });
  };
  fetchIssues = (
    workspaceSlug: string,
    userID: string,
    _loader: TLoader,
    options: IssueListOptions,
    view: TProfileViews
  ) => {
    const workspaceID = this.rootIssueStore.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
    if (!workspaceID) return Promise.resolve(undefined);
    this.setViewId(view);
    const filters = this.issueFilterStore.getIssueFilters(userID);
    const expression = filters?.richFilters ?? {};
    const query = {
      ...memberListQuery(toMemberFilters(expression), filters?.displayFilters, view),
      limit: options.limit,
    };
    return this.select({ scope: "member", workspaceSlug, workspaceID, userID, query }, options).fetchFirst();
  };
  fetchNextIssues = (slug: string, userID: string, group?: string) => {
    const list = this.currentList;
    return list?.scope.scope === "member" && list.scope.workspaceSlug === slug && list.scope.userID === userID
      ? list.fetchNext(group)
      : Promise.resolve(undefined);
  };
  fetchIssuesWithExistingPagination = (slug: string, user: string, loader: TLoader) =>
    this.options ? this.fetchIssues(slug, user, loader, this.options, this.currentView) : Promise.resolve(undefined);
}
