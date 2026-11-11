/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueCreate, IssueDetail } from "@nerve/api-client";
import type { TLoader } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { IssueListBinding } from "../helpers/issue-list-binding";
import type { IPagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import { PagePreferencesIssues } from "../helpers/page-preferences-issues.store";

/** A cycle's page, as its route params name it: the cycle and the project it belongs to. */
export type CyclePage = Readonly<{ projectID: string; cycleID: string }>;
export type ICycleIssues = CycleIssues;

/**
 * The cycle page's list (P5b spec 2.1, 2.5): its project's list under the cycle page's preferences and the term
 * `cycle_id`, so each cycle has its own binding. The page passes its cycle on each call; nothing follows the route.
 */
export class CycleIssues extends PagePreferencesIssues<IPagePreferencesFilter> {
  /** The current list when it is the page's (in `slug` when given). */
  private pageList(page: CyclePage, slug?: string): IssueListBinding | undefined {
    const list = this.currentList;
    const scope = list?.scope;
    return scope?.scope === "project" &&
      scope.projectID === page.projectID &&
      (slug === undefined || scope.workspaceSlug === slug) &&
      scope.query.cycle_id?.length === 1 &&
      scope.query.cycle_id[0] === page.cycleID
      ? list
      : undefined;
  }
  /** Compare only preference-derived terms; pagination and layout-specific options remain independent. */
  isPreferenceQueryCurrent(page: CyclePage): boolean {
    return !!this.pageList(page) && this.isPreferenceSelected(page.cycleID);
  }
  /**
   * The page's work items as nerve counts them (P5b spec 2.6), while the current list is the page's: until the page's
   * list is selected the current one may be another cycle's, whose count is not this page's (G19).
   */
  pageTotal(page: CyclePage): number | undefined {
    return this.pageList(page)?.value?.total;
  }
  fetchIssues = (slug: string, page: CyclePage, _loader: TLoader, options: IssueListOptions) =>
    this.fetchPage(slug, page.projectID, page.cycleID, options, { cycle_id: [page.cycleID] });
  fetchNextIssues = (slug: string, page: CyclePage, group?: string, subgroup?: string) =>
    this.pageList(page, slug)?.fetchNext(group, subgroup) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, page: CyclePage, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, page, loader, this.options) : Promise.resolve(undefined);
  /**
   * A creation in the cycle's project joins the cycle in its one request (R4). Another project has no such cycle:
   * nerve would refuse it with a `cycle_id` error the form cannot show before P7, so it goes without (re-66 M3).
   */
  override createIssue = (slug: string, project: string, body: IssueCreate, page?: CyclePage): Promise<IssueDetail> =>
    this.rootIssueStore.createIssue(
      this.rootIssueStore.scope(slug, project),
      page && project === page.projectID ? { ...body, cycle_id: page.cycleID } : body
    );
}
