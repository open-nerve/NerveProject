/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { IssueListBinding } from "../helpers/issue-list-binding";
import { PagePreferencesIssues } from "../helpers/page-preferences-issues.store";
import type { IProjectIssuesFilter } from "./filter.store";
export type IProjectIssues = ProjectIssues;
/** The project page's list: its project's list under the project page's preferences. */
export class ProjectIssues extends PagePreferencesIssues<IProjectIssuesFilter> {
  /** The current list when it is the project's (in `slug` when given). */
  private pageList(projectID: string, slug?: string): IssueListBinding | undefined {
    const list = this.currentList;
    const scope = list?.scope;
    return scope?.scope === "project" &&
      scope.projectID === projectID &&
      (slug === undefined || scope.workspaceSlug === slug)
      ? list
      : undefined;
  }
  /** Compare only preference-derived terms; pagination and layout-specific options remain independent. */
  isPreferenceQueryCurrent(projectID: string): boolean {
    return !!this.pageList(projectID) && this.isPreferenceSelected(projectID);
  }
  /**
   * The project's work items as nerve counts them, grouped or not (P7c spec 2.8, R10), while the current list is the
   * project's: until the project's list is selected the current one may be another project's, whose count is not this
   * page's.
   */
  pageTotal(projectID: string): number | undefined {
    return this.pageList(projectID)?.value?.total;
  }
  fetchIssues = (slug: string, projectID: string, _loader: TLoader, options: IssueListOptions) =>
    this.fetchPage(slug, projectID, projectID, options);
  fetchNextIssues = (slug: string, projectID: string, group?: string, subgroup?: string) =>
    this.pageList(projectID, slug)?.fetchNext(group, subgroup) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, project: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, project, loader, this.options) : Promise.resolve(undefined);
}
