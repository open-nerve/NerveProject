/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { intersectTargetDates, projectListQuery, toProjectFilters, type IssueListOptions } from "@/lib/issue-filters";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { BaseIssuesStore } from "./base-issues.store";
import type { IPagePreferencesFilter } from "./page-preferences-filter.store";

/**
 * A list page of one project whose query is the page's own preferences (P5b spec 2.1, R1): the project's page, and a
 * cycle's page that adds its cycle. The binding's scope is the project's either way; the page's terms are in its query.
 */
export class PagePreferencesIssues<TFilter extends IPagePreferencesFilter> extends BaseIssuesStore<TFilter> {
  viewFlags = { enableQuickAdd: true, enableIssueCreation: true, enableInlineEditing: true };
  private selectedPreferenceQuery: string | undefined;

  private preferenceQuery(preferencesID: string): ProjectIssueQuery {
    const filters = this.issueFilterStore.getIssueFilters(preferencesID);
    return projectListQuery(toProjectFilters(filters?.richFilters ?? {}), filters?.displayFilters);
  }

  /** Whether the last read was made with the page's preferences as they are now; the caller knows the page is current. */
  protected isPreferenceSelected(preferencesID: string): boolean {
    return this.selectedPreferenceQuery === JSON.stringify(this.preferenceQuery(preferencesID));
  }

  /**
   * Reads the first page of the project's list under the page's preferences and the layout's options, with the page's
   * own `terms` last so that no option replaces them. Pagination and layout options stay out of the preference check.
   */
  protected fetchPage(
    slug: string,
    projectID: string,
    preferencesID: string,
    options: IssueListOptions,
    terms: ProjectIssueQuery = {}
  ) {
    const scope = this.rootIssueStore.scope(slug, projectID);
    const preferenceQuery = this.preferenceQuery(preferencesID);
    const query = { ...preferenceQuery, limit: options.limit, ...options.query, ...terms };
    const intersection = intersectTargetDates(preferenceQuery.target_date, options.query?.target_date);
    query.target_date = intersection.target_date;
    this.selectedPreferenceQuery = JSON.stringify(preferenceQuery);
    return this.select({ ...scope, scope: "project", query, empty: intersection.empty }, options).fetchFirst();
  }
}
