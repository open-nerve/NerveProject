/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueCreate, IssueDetail } from "@nerve/api-client";
import type { IPagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import { PlanningPageIssues, type PlanningPage } from "../helpers/planning-page-issues.store";
import type { IIssueRootStore } from "../root.store";

export type ICycleIssues = CycleIssues;

/**
 * The cycle page's list (P5b spec 2.1, 2.5; P6b spec 2.2): the planning page's list by the term `cycle_id`, so each
 * cycle has its own binding; what is the cycle's own is that a creation joins it.
 */
export class CycleIssues extends PlanningPageIssues {
  constructor(rootIssueStore: IIssueRootStore, issueFilterStore: IPagePreferencesFilter) {
    super(rootIssueStore, issueFilterStore, "cycle_id");
  }
  /**
   * A creation in the cycle's project joins the cycle in its one request (R4). Another project has no such cycle:
   * nerve would refuse it with a `cycle_id` error the form cannot show before P7, so it goes without (re-66 M3).
   */
  override createIssue = (
    slug: string,
    project: string,
    body: IssueCreate,
    page?: PlanningPage
  ): Promise<IssueDetail> =>
    this.rootIssueStore.writes.createIssue(
      this.rootIssueStore.scope(slug, project),
      page && project === page.projectID ? { ...body, cycle_id: page.entityID } : body
    );
}
