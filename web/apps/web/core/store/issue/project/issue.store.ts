/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TLoader } from "@nerve/types";
import { projectListQuery, toProjectFilters, type IssueListOptions } from "@/lib/issue-filters";
import { BaseIssuesStore } from "../helpers/base-issues.store";
import type { IProjectIssuesFilter } from "./filter.store";
export type IProjectIssues = ProjectIssues;
export class ProjectIssues extends BaseIssuesStore<IProjectIssuesFilter> {
  viewFlags = { enableQuickAdd: true, enableIssueCreation: true, enableInlineEditing: true };
  private selectedPreferenceQuery: string | undefined;
  private preferenceQuery(projectID: string) {
    const filters = this.issueFilterStore.getIssueFilters(projectID);
    return projectListQuery(toProjectFilters(filters?.richFilters ?? {}), filters?.displayFilters);
  }
  /** Compare only preference-derived terms; pagination and layout-specific options remain independent. */
  isPreferenceQueryCurrent(projectID: string): boolean {
    return (
      this.currentList?.scope.scope === "project" &&
      this.currentList.scope.projectID === projectID &&
      this.selectedPreferenceQuery === JSON.stringify(this.preferenceQuery(projectID))
    );
  }
  fetchIssues = (slug: string, projectID: string, _loader: TLoader, options: IssueListOptions) => {
    const scope = this.rootIssueStore.scope(slug, projectID);
    const preferenceQuery = this.preferenceQuery(projectID);
    const query = {
      ...preferenceQuery,
      limit: options.limit,
      ...options.query,
    };
    this.selectedPreferenceQuery = JSON.stringify(preferenceQuery);
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
