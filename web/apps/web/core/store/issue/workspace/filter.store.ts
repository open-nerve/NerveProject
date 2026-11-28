/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { remove, runInAction } from "mobx";
import type { View, ViewFilters } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { toWorkspaceFilters, type WorkspacePageFilters } from "@/lib/issue-filters";
import {
  PagePreferencesFilter,
  type PagePreferences,
  type PagePreferencesService,
} from "../helpers/page-preferences-filter.store";
import type { IIssueRootStore } from "../root.store";
import { ViewDraftService } from "../helpers/view-draft.service";
import { DefaultViewPreferences } from "./default-view-preferences";
import { defaultViewOfPage } from "./page-id";

/** A workspace's view's filters as its page holds them: a valid one never has a cycle or a module (M7/P6b spec 2.4). */
const workspaceViewFilters = ({
  cycle_id: _cycle,
  module_id: _module,
  ...filters
}: ViewFilters): WorkspacePageFilters => filters;

/** A workspace's pages' preferences, by page id: a default view's the browser's, a saved view's its draft of the view. */
class WorkspacePagePreferences implements PagePreferencesService<WorkspacePageFilters> {
  constructor(
    readonly defaults: DefaultViewPreferences,
    readonly drafts: ViewDraftService<WorkspacePageFilters>
  ) {}

  private of(id: string) {
    return defaultViewOfPage(id) ? this.defaults : this.drafts;
  }

  get(id: string) {
    return this.of(id).get(id);
  }

  update = (id: string, body: Partial<PagePreferences<WorkspacePageFilters>>) => this.of(id).update(id, body);

  discard(id: string): void {
    this.of(id).discard(id);
  }
}

/**
 * A workspace's view's page's preferences (M7/P6b spec 2.4, R2, R3, R5): by page id (page-id.ts), read and changed
 * through the page-preferences base in the workspace's words, the contract's 11. A default view's page keeps its
 * display in the browser and its filters for one entry; a saved view's page is its draft of the view, as a project's
 * view's page's is (P6a): nothing of either is sent ("Update view" saves a draft). The page ends its preferences as it
 * leaves.
 */
export class GlobalIssuesFilter extends PagePreferencesFilter<WorkspacePageFilters> {
  private readonly pages: WorkspacePagePreferences;

  constructor(rootIssueStore: IIssueRootStore, readView: (viewId: string) => Promise<View | null | undefined>) {
    const pages = new WorkspacePagePreferences(
      new DefaultViewPreferences(() => rootIssueStore.currentUserId),
      new ViewDraftService(readView, workspaceViewFilters)
    );
    super(rootIssueStore, pages, toWorkspaceFilters, EIssuesStoreType.GLOBAL);
    this.pages = pages;
  }

  /** Ends the page's preferences of this entry, a default view's filters or a saved view's draft, and its filters shown. */
  discard(id: string): void {
    this.pages.discard(id);
    runInAction(() => remove(this.filters, id));
  }

  /** Takes nerve's answer to Update view as the view's draft, only while the page holds the draft (re-96 I2). */
  adopt(viewId: string, view: View): void {
    this.pages.drafts.adopt(viewId, view);
  }
}
