/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import { toWorkspaceFilters, workspaceListQuery, type IssueListOptions } from "@/lib/issue-filters";
import type { WorkspaceIssueQuery } from "@/services/issue/issue-list.service";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IssueListBinding } from "../helpers/issue-list-binding";
import type { GlobalIssuesFilter } from "./filter.store";
import { ownTerm, workspacePageId } from "./page-id";

/**
 * A workspace's view's page's list (M7/P6b spec 2.3, R1): the workspace's list, of the projects the caller is a member
 * of, under the page's preferences, flat; a default view's own term last in its query (§2.5, R4). Any role of the
 * workspace reads it (issue-list-read.ts). A row is changed in place, in its own project; nothing is created from the
 * page's table. The page passes its workspace and its view on each call; nothing follows the route. Its own list, not a
 * project page's: PagePreferencesIssues builds a project's scope alone.
 */
export class GlobalIssues extends BaseIssuesStore<GlobalIssuesFilter> {
  readonly viewFlags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: true };
  private selectedPreferenceQuery: string | undefined;

  /** The page's query: its filters and its display, a default view's own term last. */
  private preferenceQuery(slug: string, globalViewId: string): WorkspaceIssueQuery {
    const filters = this.issueFilterStore.getIssueFilters(workspacePageId(slug, globalViewId));
    return {
      ...workspaceListQuery(toWorkspaceFilters(filters?.richFilters ?? {}), filters?.displayFilters),
      ...ownTerm(globalViewId),
    };
  }

  /** The current list when it is the workspace's of `slug`. */
  private pageList(slug: string): IssueListBinding | undefined {
    const list = this.currentList;
    return list?.scope.scope === "workspace" && list.scope.workspaceSlug === slug ? list : undefined;
  }

  /** Whether the current list is the workspace's, last read under the page's preferences as they are now. */
  isPreferenceQueryCurrent(slug: string, globalViewId: string): boolean {
    return (
      !!this.pageList(slug) && this.selectedPreferenceQuery === JSON.stringify(this.preferenceQuery(slug, globalViewId))
    );
  }

  /** Reads the first page of the workspace's list for the page; a workspace the store does not give, none. */
  fetchIssues = (slug: string, globalViewId: string, _loader: TLoader, options: IssueListOptions) => {
    const workspaceID = this.rootIssueStore.rootStore.workspaceRoot.getWorkspaceBySlug(slug)?.id;
    if (!workspaceID) return Promise.resolve(undefined);
    const preferenceQuery = this.preferenceQuery(slug, globalViewId);
    this.selectedPreferenceQuery = JSON.stringify(preferenceQuery);
    const query = { ...preferenceQuery, limit: options.limit };
    return this.select({ scope: "workspace", workspaceSlug: slug, workspaceID, query }, options).fetchFirst();
  };
  fetchNextIssues = (slug: string, group?: string) =>
    this.pageList(slug)?.fetchNext(group) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, globalViewId: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, globalViewId, loader, this.options) : Promise.resolve(undefined);
}
