/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import type { State } from "@nerve/api-client";
import { orderBy, isEmpty, indexOf, groupBy, set } from "lodash-es";
import { EIssueFilterType, FILTER_TO_ISSUE_MAP, ISSUE_PRIORITIES } from "@nerve/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IIssueFilterOptions,
  ISubWorkItemFilters,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
} from "@nerve/types";
import { checkDateCriteria, convertToISODateString, parseDateFilter } from "@nerve/utils";
import { issueFieldForGroup } from "@/lib/issue-filters";

const getSortOrderToFilterEmptyValues = (key: string, object: any) => {
  const value = object?.[key];

  if (typeof value !== "number" && isEmpty(value)) return 1;

  return 0;
};

// get IssueIds from Issue data List
const getIssueIds = (issues: Issue[]) => issues.map((issue) => issue?.id);

/**
 * Checks if an issue meets the date filter criteria
 * @param issue The issue to check
 * @param filterKey The date field to check ('start_date' or 'target_date')
 * @param dateFilters Array of date filter strings
 * @returns boolean indicating if the issue meets the date criteria
 */
const checkIssueDateFilter = (
  issue: Issue,
  filterKey: "start_date" | "target_date",
  dateFilters: string[]
): boolean => {
  if (!dateFilters || dateFilters.length === 0) return true;

  const issueDate = issue[filterKey];
  if (!issueDate) return false;

  // Issue should match all the date filters (AND operation)
  return dateFilters.every((filterValue) => {
    const parsed = parseDateFilter(filterValue);
    if (!parsed?.date || !parsed?.type) {
      // ignore invalid filter instead of failing the whole evaluation
      console.warn(`[filters] Ignoring unparsable date filter "${filterValue}"`);
      return true;
    }
    return checkDateCriteria(new Date(issueDate), parsed.date, parsed.type);
  });
};

/**
 * Filters the given work items based on the provided filters and display filters.
 * @param work items - The array of work items to be filtered.
 * @param filters - The filters to be applied to the issues.
 * @param displayFilters - The display filters to be applied to the issues.
 * @returns The filtered array of issues.
 */
export const getFilteredWorkItems = (
  workItems: Issue[],
  filters: IIssueFilterOptions | undefined,
  stateMap?: Record<string, State>
): Issue[] => {
  if (!filters) return workItems;
  // Get all active filters
  const activeFilters = Object.entries(filters).filter(([, value]) => value && value.length > 0);
  // If no active filters, return all issues
  if (activeFilters.length === 0) {
    return workItems;
  }

  return workItems.filter((workItem) =>
    // Check all filter conditions (AND operation between different filters)
    activeFilters.every(([filterKey, filterValues]) => {
      // Handle date filters separately
      if (filterKey === "start_date" || filterKey === "target_date") {
        return checkIssueDateFilter(workItem, filterKey, filterValues as string[]);
      }
      if (filterKey === "state_group") {
        const group = workItem.state_id ? stateMap?.[workItem.state_id]?.group : undefined;
        return group !== undefined && filterValues!.includes(group);
      }
      // Handle regular filters
      const issueKey = FILTER_TO_ISSUE_MAP[filterKey as keyof IIssueFilterOptions];
      if (!issueKey) return true; // Skip if no mapping exists
      const issueValue = workItem[issueKey];
      // Handle array-based properties vs single value properties
      if (Array.isArray(issueValue)) {
        return filterValues!.some((filterValue: any) => issueValue.includes(filterValue));
      } else {
        return filterValues!.includes(issueValue as string);
      }
    })
  );
};

/**
 * Orders the given work items based on the provided order by key.
 * @param workItems - The array of work items to be ordered.
 * @param orderByKey - The key to order the issues by.
 * @returns The ordered array of work items.
 */
const getOrderedWorkItems = (workItems: Issue[], orderByKey: TIssueOrderByOptions): string[] => {
  switch (orderByKey) {
    case "-updated_at":
      return getIssueIds(orderBy(workItems, (item) => convertToISODateString(item["updated_at"]), ["desc"]));

    case "-created_at":
      return getIssueIds(orderBy(workItems, (item) => convertToISODateString(item["created_at"]), ["desc"]));

    case "-start_date":
      return getIssueIds(
        orderBy(
          workItems,
          [getSortOrderToFilterEmptyValues.bind(null, "start_date"), "start_date"], //preferring sorting based on empty values to always keep the empty values below
          ["asc", "desc"]
        )
      );

    case "-priority": {
      const sortArray = ISSUE_PRIORITIES.map((i) => i.key);
      return getIssueIds(
        orderBy(workItems, (currentIssue: Issue) => indexOf(sortArray, currentIssue?.priority), ["asc"])
      );
    }
    default:
      return getIssueIds(workItems);
  }
};

export const getGroupedWorkItemIds = (
  workItems: Issue[],
  groupByKey?: TIssueGroupByOptions,
  orderByKey: TIssueOrderByOptions = "-created_at",
  stateMap?: Record<string, State>
): Record<string, string[]> => {
  // If group by is not set set default as ALL ISSUES
  if (!groupByKey) {
    return {
      [""]: getOrderedWorkItems(workItems, orderByKey),
    };
  }

  // State groups belong to the directory, never to the work item entity.
  const groupKey = issueFieldForGroup(groupByKey);
  const groupedWorkItems = groupBy(workItems, (item) => {
    const value =
      groupByKey === "state_detail.group"
        ? item.state_id
          ? stateMap?.[item.state_id]?.group
          : undefined
        : item[groupKey];
    if (Array.isArray(value)) {
      if (value.length === 0) return "none";
      // Sort & join to build deterministic set-like key
      return value.toSorted().join(",");
    }
    return value ?? "none";
  });

  // Convert to Record type
  const groupedWorkItemsRecord: Record<string, string[]> = {};
  Object.entries(groupedWorkItems).forEach(([key, items]) => {
    groupedWorkItemsRecord[key] = getOrderedWorkItems(items, orderByKey);
  });

  return groupedWorkItemsRecord;
};

/**
 * Updates the filters for a given work item.
 * @param filtersMap - The map of filters for the work item.
 * @param filterType - The type of filter to update.
 * @param filters - The filters to update.
 * @param workItemId - The ID of the work item to update.
 */
export const updateSubWorkItemFilters = (
  filtersMap: Record<string, Partial<ISubWorkItemFilters>>,
  filterType: EIssueFilterType,
  filters: IIssueDisplayFilterOptions | IIssueDisplayProperties | IIssueFilterOptions,
  workItemId: string
) => {
  const existingFilters = filtersMap[workItemId] ?? {};
  const _filters = {
    filters: existingFilters.filters,
    displayFilters: existingFilters.displayFilters,
    displayProperties: existingFilters.displayProperties,
  };

  switch (filterType) {
    case EIssueFilterType.FILTERS: {
      const updatedFilters = filters as IIssueFilterOptions;
      _filters.filters = { ..._filters.filters, ...updatedFilters };
      set(filtersMap, [workItemId, "filters"], { ..._filters.filters, ...updatedFilters });
      break;
    }
    case EIssueFilterType.DISPLAY_FILTERS: {
      set(filtersMap, [workItemId, "displayFilters"], { ..._filters.displayFilters, ...filters });
      break;
    }
    case EIssueFilterType.DISPLAY_PROPERTIES:
      set(filtersMap, [workItemId, "displayProperties"], {
        ..._filters.displayProperties,
        ...filters,
      });
      break;
  }
};
