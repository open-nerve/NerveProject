/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import { PagePreferencesIssues } from "../helpers/page-preferences-issues.store";
import type { IProjectIssuesFilter } from "./filter.store";
export type IProjectIssues = ProjectIssues;
/** The project page's list: its project's list under the project page's preferences. */
export class ProjectIssues extends PagePreferencesIssues<IProjectIssuesFilter> {
  /** Compare only preference-derived terms; pagination and layout-specific options remain independent. */
  isPreferenceQueryCurrent(projectID: string): boolean {
    return (
      this.currentList?.scope.scope === "project" &&
      this.currentList.scope.projectID === projectID &&
      this.isPreferenceSelected(projectID)
    );
  }
  fetchIssues = (slug: string, projectID: string, _loader: TLoader, options: IssueListOptions) =>
    this.fetchPage(slug, projectID, projectID, options);
  fetchNextIssues = (slug: string, projectID: string, group?: string, subgroup?: string) => {
    const list = this.currentList;
    return list?.scope.scope === "project" && list.scope.workspaceSlug === slug && list.scope.projectID === projectID
      ? list.fetchNext(group, subgroup)
      : Promise.resolve(undefined);
  };
  fetchIssuesWithExistingPagination = (slug: string, project: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, project, loader, this.options) : Promise.resolve(undefined);
}
