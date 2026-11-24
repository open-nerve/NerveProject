/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { IssueListBinding } from "../helpers/issue-list-binding";
import { PagePreferencesIssues } from "../helpers/page-preferences-issues.store";
import type { ProjectViewIssuesFilter } from "./filter.store";

/**
 * A project's view's page's list (P6a spec 2.5, R1): its project's list under the page's draft of the view, by the
 * view's id, a list of its own beside the project page's, whose preferences it neither reads nor changes. The page
 * passes its project and its view on each call; nothing follows the route.
 */
export class ProjectViewIssues extends PagePreferencesIssues<ProjectViewIssuesFilter> {
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
  /** Whether the current list is the project's, last read under the view's draft as it is now. */
  isPreferenceQueryCurrent(projectID: string, viewID: string): boolean {
    return !!this.pageList(projectID) && this.isPreferenceSelected(viewID);
  }
  fetchIssues = (slug: string, projectID: string, viewID: string, _loader: TLoader, options: IssueListOptions) =>
    this.fetchPage(slug, projectID, viewID, options);
  fetchNextIssues = (slug: string, projectID: string, group?: string, subgroup?: string) =>
    this.pageList(projectID, slug)?.fetchNext(group, subgroup) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, projectID: string, viewID: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, projectID, viewID, loader, this.options) : Promise.resolve(undefined);
}
