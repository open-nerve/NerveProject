/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueCreate, IssueDetail } from "@nerve/api-client";
import type { IPagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import { PlanningPageIssues, type PlanningPage } from "../helpers/planning-page-issues.store";
import type { IIssueRootStore } from "../root.store";

export type IModuleIssues = ModuleIssues;

/**
 * The module page's list (P6b spec 2.3, 2.5): the planning page's list by the term `module_id`, so each module has its
 * own binding. What is the module's own: a work item is in a set of modules, which nerve replaces whole on a change, so
 * a creation joins the module to the set it is given, and a removal sends the set the work item holds without it.
 */
export class ModuleIssues extends PlanningPageIssues {
  constructor(rootIssueStore: IIssueRootStore, issueFilterStore: IPagePreferencesFilter) {
    super(rootIssueStore, issueFilterStore, "module_id");
  }
  /**
   * A creation in the module's project joins the module in its one request, beside any modules it is given (R6).
   * Another project has no such module: nerve would refuse it with a `module_ids` error the form cannot show before
   * P7, so it goes as given.
   */
  override createIssue = (
    slug: string,
    project: string,
    body: IssueCreate,
    page?: PlanningPage
  ): Promise<IssueDetail> =>
    this.rootIssueStore.writes.createIssue(
      this.rootIssueStore.scope(slug, project),
      page && project === page.projectID
        ? { ...body, module_ids: [...new Set([...(body.module_ids ?? []), page.entityID])] }
        : body
    );
  /**
   * Takes the work item out of the page's module (R5; P7a R19): the change that removes the module, which the write
   * unit makes, when its turn in the item's queue comes, into the whole set held without it, so after a write still
   * out it is that write's answer. The other modules stay, archived ones too (nerve checks only the modules a work item
   * joins). Once nerve confirms it the page's list no longer lists the work item; a refusal leaves what is held and
   * goes to the caller.
   */
  removeIssueFromModule = (slug: string, page: PlanningPage, id: string): Promise<IssueDetail> =>
    this.rootIssueStore.writes.changeIssue(
      id,
      { modules: { remove: [page.entityID] } },
      this.rootIssueStore.scope(slug, page.projectID)
    );
}
