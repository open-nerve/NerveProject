/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import type { IIssueRootStore } from "../root.store";
import type { IssueListBinding } from "./issue-list-binding";
import type { IPagePreferencesFilter } from "./page-preferences-filter.store";
import { PagePreferencesIssues } from "./page-preferences-issues.store";

/** The term of its project's list that makes the list a cycle's or a module's page's (P6b spec 2.2, 2.3). */
type PlanningPageTerm = Extract<keyof ProjectIssueQuery, "cycle_id" | "module_id">;
/** A cycle's or a module's page, as its route params name it: the project, and the cycle or module it shows. */
export type PlanningPage = Readonly<{ projectID: string; entityID: string }>;

/**
 * A cycle's or a module's page's list (P5b spec 2.1, 2.5; P6b spec 2.2): its project's list under the page's own
 * preferences (by the cycle's or module's id) and the page's term with that id, so each cycle or module has its own
 * binding. The page passes its cycle or module on each call; nothing follows the route.
 */
export class PlanningPageIssues extends PagePreferencesIssues<IPagePreferencesFilter> {
  constructor(
    rootIssueStore: IIssueRootStore,
    issueFilterStore: IPagePreferencesFilter,
    private readonly term: PlanningPageTerm
  ) {
    super(rootIssueStore, issueFilterStore);
  }
  /** The current list when it is the page's (in `slug` when given). */
  private pageList(page: PlanningPage, slug?: string): IssueListBinding | undefined {
    const list = this.currentList;
    const scope = list?.scope;
    if (scope?.scope !== "project") return undefined;
    const term = scope.query[this.term];
    return scope.projectID === page.projectID &&
      (slug === undefined || scope.workspaceSlug === slug) &&
      term?.length === 1 &&
      term[0] === page.entityID
      ? list
      : undefined;
  }
  /** Compare only preference-derived terms; pagination and layout-specific options remain independent. */
  isPreferenceQueryCurrent(page: PlanningPage): boolean {
    return !!this.pageList(page) && this.isPreferenceSelected(page.entityID);
  }
  /**
   * The page's work items as nerve counts them (P5b spec 2.6), while the current list is the page's: until the page's
   * list is selected the current one may be another cycle's or module's, whose count is not this page's (G19).
   */
  pageTotal(page: PlanningPage): number | undefined {
    return this.pageList(page)?.value?.total;
  }
  fetchIssues = (slug: string, page: PlanningPage, _loader: TLoader, options: IssueListOptions) =>
    this.fetchPage(slug, page.projectID, page.entityID, options, { [this.term]: [page.entityID] });
  fetchNextIssues = (slug: string, page: PlanningPage, group?: string, subgroup?: string) =>
    this.pageList(page, slug)?.fetchNext(group, subgroup) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, page: PlanningPage, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, page, loader, this.options) : Promise.resolve(undefined);
}
