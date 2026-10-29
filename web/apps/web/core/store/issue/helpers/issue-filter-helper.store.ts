/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
// nerve constants
import type { EIssueFilterType } from "@nerve/constants";
import type {
  EIssuesStoreType,
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IIssueFilters,
  IIssueFiltersResponse,
  TIssueKanbanFilters,
  TIssueParams,
  TStaticViewTypes,
} from "@nerve/types";
// helpers
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@nerve/utils";
// lib
import { storage } from "@/lib/local-storage";

const strings = (value: unknown): string[] =>
  Array.isArray(value) && value.every((item) => typeof item === "string") ? value : [];

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const displayGroups = new Set([
  "state",
  "priority",
  "labels",
  "created_by",
  "state_detail.group",
  "project",
  "assignees",
  "target_date",
]);
const displayOrders = new Set([
  "-created_at",
  "created_at",
  "updated_at",
  "-updated_at",
  "priority",
  "-priority",
  "sort_order",
  "state__name",
  "-state__name",
  "assignees__first_name",
  "-assignees__first_name",
  "labels__name",
  "-labels__name",
  "target_date",
  "-target_date",
  "start_date",
  "-start_date",
  "link_count",
  "-link_count",
  "sub_issues_count",
  "-sub_issues_count",
]);
const displayPropertyNames = [
  "assignee",
  "start_date",
  "due_date",
  "labels",
  "key",
  "priority",
  "state",
  "sub_issue_count",
  "link",
  "created_on",
  "updated_on",
] as const;

function localDisplayFilters(value: unknown): IIssueDisplayFilterOptions {
  if (!isRecord(value)) return {};
  const result: IIssueDisplayFilterOptions = {};
  if (typeof value.layout === "string" && ["list", "kanban", "calendar", "spreadsheet"].includes(value.layout))
    result.layout = value.layout;
  if (typeof value.order_by === "string" && displayOrders.has(value.order_by))
    result.order_by = value.order_by as IIssueDisplayFilterOptions["order_by"];
  for (const field of ["group_by", "sub_group_by"] as const) {
    const group = value[field];
    if (group === null || (typeof group === "string" && displayGroups.has(group)))
      result[field] = group as IIssueDisplayFilterOptions[typeof field];
  }
  for (const field of ["show_empty_groups", "sub_issue"] as const)
    if (typeof value[field] === "boolean") result[field] = value[field];
  if (isRecord(value.calendar)) {
    result.calendar = {};
    if (value.calendar.layout === "month" || value.calendar.layout === "week")
      result.calendar.layout = value.calendar.layout;
    if (typeof value.calendar.show_weekends === "boolean") result.calendar.show_weekends = value.calendar.show_weekends;
  }
  return result;
}
function localDisplayProperties(value: unknown): IIssueDisplayProperties {
  if (!isRecord(value)) return {};
  const result: IIssueDisplayProperties = {};
  for (const field of displayPropertyNames) if (typeof value[field] === "boolean") result[field] = value[field];
  return result;
}

interface ILocalStoreIssueFilters {
  key: EIssuesStoreType;
  workspaceSlug: string;
  viewId: string | undefined; // It can be projectId, moduleId, cycleId, projectViewId
  userId: string | undefined;
  filters: Partial<
    IIssueFiltersResponse & { kanban_filters: TIssueKanbanFilters; richFilters: IIssueFiltersResponse["rich_filters"] }
  >;
}

export interface IBaseIssueFilterStore {
  // observables
  filters: Record<string, IIssueFilters>;
  //computed
  issueFilters: IIssueFilters | undefined;
}

export interface IIssueFilterHelperStore {
  computedIssueFilters(filters: IIssueFilters): IIssueFilters;
  getFilterConditionBasedOnViews: (
    currentUserId: string | undefined,
    type: TStaticViewTypes
  ) => Partial<Record<TIssueParams, string>> | undefined;
  computedDisplayFilters(
    displayFilters: IIssueDisplayFilterOptions,
    defaultValues?: IIssueDisplayFilterOptions
  ): IIssueDisplayFilterOptions;
  computedDisplayProperties(filters: IIssueDisplayProperties): IIssueDisplayProperties;
}

export class IssueFilterHelperStore implements IIssueFilterHelperStore {
  /**
   * @description This method is used to apply the display filters on the issues
   * @param {IIssueFilters} filters
   * @returns {IIssueFilters}
   */
  computedIssueFilters = (filters: IIssueFilters): IIssueFilters => ({
    richFilters: isEmpty(filters?.richFilters) ? {} : filters?.richFilters,
    displayFilters: isEmpty(filters?.displayFilters) ? undefined : filters?.displayFilters,
    displayProperties: isEmpty(filters?.displayProperties) ? undefined : filters?.displayProperties,
    kanbanFilters: isEmpty(filters?.kanbanFilters) ? undefined : filters?.kanbanFilters,
  });

  /**
   * @description This method is used to get the filter conditions based on the views
   * @param currentUserId
   * @param type
   * @returns
   */
  getFilterConditionBasedOnViews: IIssueFilterHelperStore["getFilterConditionBasedOnViews"] = (currentUserId, type) => {
    if (!currentUserId) return undefined;
    switch (type) {
      case "assigned":
        return {
          assignees: currentUserId,
        };
      case "created":
        return {
          created_by: currentUserId,
        };
      case "subscribed":
        return {
          subscriber: currentUserId,
        };
      case "all-issues":
      default:
        return undefined;
    }
  };

  /**
   * @description This method is used to apply the display filters on the issues
   * @param {IIssueDisplayFilterOptions} displayFilters
   * @returns {IIssueDisplayFilterOptions}
   */
  computedDisplayFilters = (
    displayFilters: IIssueDisplayFilterOptions,
    defaultValues?: IIssueDisplayFilterOptions
  ): IIssueDisplayFilterOptions => {
    const computedFilters = getComputedDisplayFilters(displayFilters, defaultValues);
    return computedFilters;
  };

  /**
   * @description This method is used to apply the display properties on the issues
   * @param {IIssueDisplayProperties} displayProperties
   * @returns {IIssueDisplayProperties}
   */
  computedDisplayProperties = (displayProperties: IIssueDisplayProperties): IIssueDisplayProperties =>
    getComputedDisplayProperties(displayProperties);

  handleIssuesLocalFilters = {
    fetchFiltersFromStorage: () => {
      try {
        const raw = storage.get("issue_local_filters");
        const parsed: unknown = raw ? JSON.parse(raw) : [];
        if (!Array.isArray(parsed)) return [];
        return parsed.filter(
          (entry): entry is ILocalStoreIssueFilters =>
            entry !== null &&
            typeof entry === "object" &&
            !Array.isArray(entry) &&
            entry.filters !== null &&
            typeof entry.filters === "object" &&
            !Array.isArray(entry.filters)
        );
      } catch {
        return [];
      }
    },

    get: (
      currentView: EIssuesStoreType,
      workspaceSlug: string,
      viewId: string | undefined, // It can be projectId, moduleId, cycleId, projectViewId
      userId: string | undefined
    ) => {
      if (!userId) return undefined;
      const storageFilters = this.handleIssuesLocalFilters.fetchFiltersFromStorage();
      const currentFilterIndex = storageFilters.findIndex(
        (filter: ILocalStoreIssueFilters) =>
          filter.key === currentView &&
          filter.workspaceSlug === workspaceSlug &&
          filter.viewId === viewId &&
          filter.userId === userId
      );
      const entry = storageFilters[currentFilterIndex];
      if (!entry) return undefined;
      if ("richFilters" in entry.filters) {
        const { richFilters, ...rest } = entry.filters;
        entry.filters = { ...rest, rich_filters: rest.rich_filters ?? richFilters };
        try {
          storage.set("issue_local_filters", JSON.stringify(storageFilters));
        } catch {
          /* storage unavailable */
        }
      }
      const kanban = entry.filters.kanban_filters;
      return {
        ...entry.filters,
        display_filters: localDisplayFilters(entry.filters.display_filters),
        display_properties: localDisplayProperties(entry.filters.display_properties),
        ...(kanban
          ? { kanban_filters: { group_by: strings(kanban.group_by), sub_group_by: strings(kanban.sub_group_by) } }
          : {}),
      };
    },

    set: (
      currentView: EIssuesStoreType,
      filterType: EIssueFilterType,
      workspaceSlug: string,
      viewId: string | undefined, // It can be projectId, moduleId, cycleId, projectViewId
      userId: string | undefined,
      filters: Partial<IIssueFiltersResponse & { kanban_filters: TIssueKanbanFilters }>
    ) => {
      if (!userId) return;
      const storageFilters = this.handleIssuesLocalFilters.fetchFiltersFromStorage();
      const currentFilterIndex = storageFilters.findIndex(
        (filter: ILocalStoreIssueFilters) =>
          filter.key === currentView &&
          filter.workspaceSlug === workspaceSlug &&
          filter.viewId === viewId &&
          filter.userId === userId
      );

      const existing = storageFilters[currentFilterIndex];
      if (existing && "richFilters" in existing.filters) {
        const { richFilters, ...rest } = existing.filters;
        existing.filters = { ...rest, rich_filters: rest.rich_filters ?? richFilters };
      }
      if (currentFilterIndex < 0)
        storageFilters.push({
          key: currentView,
          workspaceSlug: workspaceSlug,
          viewId: viewId,
          userId: userId,
          filters: filters,
        });
      else
        storageFilters[currentFilterIndex] = {
          ...storageFilters[currentFilterIndex],
          filters: {
            ...storageFilters[currentFilterIndex].filters,
            [filterType]: filters[filterType as keyof IIssueFiltersResponse],
          },
        };
      // All group_by "filters" are stored in a single array, will cause inconsistency in case of duplicated values
      try {
        storage.set("issue_local_filters", JSON.stringify(storageFilters));
      } catch {
        /* storage unavailable */
      }
    },
  };

  /**
   * This Method returns true if the display properties changed requires a server side update
   * @param displayFilters
   * @returns
   */
  getShouldReFetchIssues = (displayFilters: IIssueDisplayFilterOptions) => {
    const NON_SERVER_DISPLAY_FILTERS = ["order_by", "sub_issue", "type"];
    const displayFilterKeys = Object.keys(displayFilters);

    return NON_SERVER_DISPLAY_FILTERS.some((serverDisplayfilter: string) =>
      displayFilterKeys.includes(serverDisplayfilter)
    );
  };

  getShouldClearIssues = (display: IIssueDisplayFilterOptions) => Object.hasOwn(display, "layout");
}
