/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import { projectListQuery, toProjectFilters, type IssueListOptions } from "@/lib/issue-filters";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IProjectIssuesFilter } from "./filter.store";
export type IProjectIssues = ProjectIssues;
export class ProjectIssues extends BaseIssuesStore<IProjectIssuesFilter> {
  viewFlags = { enableQuickAdd: true, enableIssueCreation: true, enableInlineEditing: true };
  fetchIssues = (slug: string, projectID: string, _loader: TLoader, options: IssueListOptions) => {
    const scope = this.rootIssueStore.scope(slug, projectID);
    const filters = this.issueFilterStore.getIssueFilters(projectID);
    const query = {
      ...projectListQuery(toProjectFilters(filters?.richFilters ?? {}), filters?.displayFilters),
      limit: options.limit,
      ...options.query,
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
