/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { clone, isNil } from "lodash-es";
// nerve types
import type {
  IIssueDisplayProperties,
  TIssueGroupByOptions,
  IIssueFilterOptions,
  IIssueFilters,
  IIssueDisplayFilterOptions,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";

// store
import { DEFAULT_DISPLAY_PROPERTIES } from "@/store/issue/issue-details/sub_issues_filter.store";
// constants
import { ISSUE_GROUP_BY_OPTIONS } from "@nerve/constants";

export const HIGHLIGHT_CLASS = "highlight";

export const isWorkspaceLevel = (type: EIssuesStoreType) =>
  [EIssuesStoreType.PROFILE, EIssuesStoreType.WORKSPACE_DRAFT].includes(type);

export const getDisplayPropertiesCount = (
  displayProperties: IIssueDisplayProperties,
  ignoreFields?: (keyof IIssueDisplayProperties)[]
) => {
  const propertyKeys = Object.keys(displayProperties) as (keyof IIssueDisplayProperties)[];

  let count = 0;

  for (const propertyKey of propertyKeys) {
    if (ignoreFields && ignoreFields.includes(propertyKey)) continue;
    if (displayProperties[propertyKey]) count++;
  }

  return count;
};

/**
 * Returns Sort order of the issue block at the position of drop
 * @param destinationIssues
 * @param destinationIssueId
 * @param getIssueById
 * @returns
 */
export const getIssueBlockId = (issueId: string | undefined, groupId: string | undefined, subGroupId?: string) =>
  `issue_${issueId}_${groupId}_${subGroupId}`;

/**
 * method that removes Null or undefined Keys from object
 * @param obj
 * @returns
 */
export const removeNillKeys = <T,>(obj: T) =>
  Object.fromEntries(Object.entries(obj ?? {}).filter(([key, value]) => key && !isNil(value)));

/**
 * This Method returns if the issue is new or not
 * @param issue
 * @returns
 */
export const isIssueNew = (issue: Issue) => {
  const createdDate = new Date(issue.created_at);
  const currentDate = new Date();
  const diff = currentDate.getTime() - createdDate.getTime();
  return diff < 30000;
};

/**
 * Returns approximate height of Kanban card based on display properties
 * @param displayProperties
 * @returns
 */
export function getApproximateCardHeight(displayProperties: IIssueDisplayProperties | undefined) {
  if (!displayProperties) return 100;

  // default card height
  let cardHeight = 46;

  const clonedProperties = clone(displayProperties);

  // key adds the height for key
  if (clonedProperties.key) {
    cardHeight += 24;
  }

  // Ignore smaller dimension properties
  const ignoredProperties: (keyof IIssueDisplayProperties)[] = [
    "key",
    "sub_issue_count",
    "link",
    "attachment_count",
    "created_on",
    "updated_on",
  ];

  ignoredProperties.forEach((key: keyof IIssueDisplayProperties) => {
    delete clonedProperties[key];
  });

  let propertyCount = 0;

  // count the remaining properties
  (Object.keys(clonedProperties) as (keyof IIssueDisplayProperties)[]).forEach((key: keyof IIssueDisplayProperties) => {
    if (clonedProperties[key]) {
      propertyCount++;
    }
  });

  // based on property count, approximate the height of each card
  if (propertyCount > 3) {
    cardHeight += 60;
  } else if (propertyCount > 0) {
    cardHeight += 32;
  }

  return cardHeight;
}

/**
 * This method returns if the filters are applied
 * @param filters
 * @returns
 */
export const isDisplayFiltersApplied = (filters: Partial<IIssueFilters>): boolean => {
  const isDisplayPropertiesApplied = Object.keys(DEFAULT_DISPLAY_PROPERTIES).some(
    (key) => !filters.displayProperties?.[key as keyof IIssueDisplayProperties]
  );

  const changedDisplayFilters = Object.keys(filters.displayFilters ?? {}).some((key) => {
    const value = filters.displayFilters?.[key as keyof IIssueDisplayFilterOptions];
    if (!value) return false;
    // -create_at is the default order
    if (key === "order_by") {
      return value !== "-created_at";
    }
    return true;
  });

  return isDisplayPropertiesApplied || changedDisplayFilters;
};

/**
 * This method returns if the filters are applied
 * @param filters
 * @returns
 */
export const isFiltersApplied = (filters: IIssueFilterOptions): boolean =>
  Object.values(filters).some((value) => {
    if (Array.isArray(value)) return value.length > 0;
    return value !== undefined && value !== null && value !== "";
  });

/**
 * Calculates the minimum width needed for issue identifiers in list layouts
 * @param projectIdentifierLength - Length of the project identifier (e.g., "PROJ" = 4)
 * @param sequenceId - Actual sequence ID of a displayed work item (e.g., 1234)
 * @returns Width in pixels needed to display the identifier
 *
 * @example
 * // For "PROJ-1234"
 * calculateIdentifierWidth(4, 1234) // Returns width for "PROJ" + "-" + "1234"
 */
export const calculateIdentifierWidth = (projectIdentifierLength: number, sequenceId: number): number => {
  const sequenceDigits = Math.max(1, Math.floor(Math.log10(sequenceId)) + 1);
  return projectIdentifierLength * 7 + 7 + sequenceDigits * 7; // project identifier chars + dash + sequence digits
};

export const useGroupByOptions = (
  options: TIssueGroupByOptions[]
): {
  key: TIssueGroupByOptions;
  titleTranslationKey: string;
}[] => {
  const groupByOptions = ISSUE_GROUP_BY_OPTIONS.filter((option) => options.includes(option.key));
  return groupByOptions;
};
