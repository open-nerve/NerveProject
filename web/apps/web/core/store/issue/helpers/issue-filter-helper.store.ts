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
