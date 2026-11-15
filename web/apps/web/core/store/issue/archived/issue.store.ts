/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import { projectListQuery, toProjectFilters, type IssueListOptions } from "@/lib/issue-filters";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IssueListBinding } from "../helpers/issue-list-binding";
import type { IArchivedIssuesFilter } from "./filter.store";
export type IArchivedIssues = ArchivedIssues;
export class ArchivedIssues extends BaseIssuesStore<IArchivedIssuesFilter> {
  viewFlags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: false };
  /** The current list when it is the project's archived work items' (in `slug` when given). */
  private pageList(projectID: string, slug?: string): IssueListBinding | undefined {
    const list = this.currentList;
    const scope = list?.scope;
    return scope?.scope === "project" &&
      scope.projectID === projectID &&
      (slug === undefined || scope.workspaceSlug === slug)
      ? list
      : undefined;
  }
  /**
   * The project's archived work items as nerve counts them, grouped or not (P7c spec 2.8, R10), while the current list
   * is the project's: until the project's list is selected the current one may be another project's.
   */
  pageTotal(projectID: string): number | undefined {
    return this.pageList(projectID)?.value?.total;
  }
  fetchIssues = (slug: string, projectID: string, _loader: TLoader, options: IssueListOptions) => {
    const scope = this.rootIssueStore.scope(slug, projectID);
    const filters = this.issueFilterStore.getIssueFilters(projectID, slug);
    const query = {
      ...projectListQuery(toProjectFilters(filters?.richFilters ?? {}), filters?.displayFilters),
      limit: options.limit,
      ...options.query,
      archived: "true",
    };
    return this.select({ ...scope, scope: "project", query }, options).fetchFirst();
  };
  fetchNextIssues = (slug: string, projectID: string, group?: string, subgroup?: string) =>
    this.pageList(projectID, slug)?.fetchNext(group, subgroup) ?? Promise.resolve(undefined);
  fetchIssuesWithExistingPagination = (slug: string, project: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, project, loader, this.options) : Promise.resolve(undefined);
}
