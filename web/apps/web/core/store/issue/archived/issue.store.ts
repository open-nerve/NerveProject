/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import { projectListQuery, toProjectFilters, type IssueListOptions } from "@/lib/issue-filters";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IArchivedIssuesFilter } from "./filter.store";
export type IArchivedIssues = ArchivedIssues;
export class ArchivedIssues extends BaseIssuesStore<IArchivedIssuesFilter> {
  viewFlags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: false };
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
  fetchNextIssues = (slug: string, projectID: string, group?: string, subgroup?: string) => {
    const list = this.currentList;
    return list?.scope.scope === "project" && list.scope.workspaceSlug === slug && list.scope.projectID === projectID
      ? list.fetchNext(group, subgroup)
      : Promise.resolve(undefined);
  };
  fetchIssuesWithExistingPagination = (slug: string, project: string, loader: TLoader = "mutation") =>
    this.options ? this.fetchIssues(slug, project, loader, this.options) : Promise.resolve(undefined);
}
