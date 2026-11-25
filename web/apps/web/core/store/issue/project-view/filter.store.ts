/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { remove, runInAction } from "mobx";
import type { ProjectIssueFilters, View } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { toProjectFilters } from "@/lib/issue-filters";
import { PagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import type { IIssueRootStore } from "../root.store";
import type { ViewDraftService } from "./view-draft.service";

/**
 * A project's view's page's preferences (P6a spec 2.5, 2.6, R1, R2): the page's draft of the view, by the view's id,
 * read and changed through the page-preferences base as the project page's preferences are, in the project page's
 * words; nothing of it is sent (R10: "Update view" saves it). The page ends it as it leaves.
 */
export class ProjectViewIssuesFilter extends PagePreferencesFilter<ProjectIssueFilters> {
  constructor(
    rootIssueStore: IIssueRootStore,
    private readonly drafts: ViewDraftService
  ) {
    super(rootIssueStore, drafts, toProjectFilters, EIssuesStoreType.PROJECT_VIEW);
  }

  /**
   * Ends the view's draft, any seed of it out, and the page's filters shown from it: entering the view again shows
   * nothing of the old draft until the view is read again (§2.6).
   */
  discard(viewID: string): void {
    this.drafts.discard(viewID);
    runInAction(() => remove(this.filters, viewID));
  }

  /** Takes nerve's answer to Update view as the view's draft (§2.7), only while the page holds the draft (re-96 I2). */
  adopt(viewID: string, view: View): void {
    this.drafts.adopt(viewID, view);
  }
}
