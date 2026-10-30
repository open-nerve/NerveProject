/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueUpdate } from "@nerve/api-client";

import { clone, isNil } from "lodash-es";
import { extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import type { FC } from "react";
import {
  CalendarOutline,
  DueDateOutline,
  LabelsOutline,
  LinkOutline,
  MembersOutline,
  PriorityOutline,
  StartDateOutline,
  StateOutline,
  WorkItemsOutline,
} from "@makeplane/propel/icons";
// nerve types
import type { ISvgIcons } from "@nerve/propel/icons";
import type {
  IIssueDisplayProperties,
  IPragmaticDropPayload,
  TIssueGroupByOptions,
  IIssueFilterOptions,
  IIssueFilters,
  IIssueDisplayFilterOptions,
  TSpreadsheetColumn,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";

// store
import type { IssueDrop } from "@/store/issue/root.store";
import { DEFAULT_DISPLAY_PROPERTIES } from "@/store/issue/issue-details/sub_issues_filter.store";
// constants
import { ISSUE_GROUP_BY_OPTIONS } from "@nerve/constants";
// components
import {
  SpreadsheetAssigneeColumn,
  SpreadsheetCreatedOnColumn,
  SpreadsheetDueDateColumn,
  SpreadsheetLabelColumn,
  SpreadsheetLinkColumn,
  SpreadsheetPriorityColumn,
  SpreadsheetStartDateColumn,
  SpreadsheetStateColumn,
  SpreadsheetSubIssueColumn,
  SpreadsheetUpdatedOnColumn,
} from "@/components/issues/issue-layouts/spreadsheet/columns";

export const HIGHLIGHT_CLASS = "highlight";

export type GroupDropLocation = {
  columnId: string;
  groupId: string;
  subGroupId?: string;
  id: string | undefined;
  canAddIssueBelow?: boolean;
};

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
 * get Kanban Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getSourceFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location, source: sourceIssue } = payload;

  const sourceIssueData = sourceIssue.data;
  let sourceColumData;

  const sourceDropTargets = location?.initial?.dropTargets ?? [];
  for (const dropTarget of sourceDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN") {
      sourceColumData = dropTargetData;
    }
  }

  if (
    typeof sourceIssueData?.id !== "string" ||
    !sourceIssueData.id ||
    typeof sourceColumData?.groupId !== "string" ||
    (sourceColumData.subGroupId !== undefined && typeof sourceColumData.subGroupId !== "string")
  )
    return;

  return {
    groupId: sourceColumData.groupId,
    subGroupId: sourceColumData.subGroupId as string | undefined,
    columnId: `${sourceColumData.groupId}__${sourceColumData.subGroupId}`,
    id: sourceIssueData.id,
  };
};

/**
 * get Destination Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getDestinationFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location } = payload;

  let destinationIssueData, destinationColumnData;

  const destDropTargets = location?.current?.dropTargets ?? [];

  for (const dropTarget of destDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN" || dropTargetData.type === "DELETE") {
      destinationColumnData = dropTargetData;
    }

    if (dropTargetData.type === "ISSUE") {
      destinationIssueData = dropTargetData;
    }
  }

  if (
    typeof destinationColumnData?.groupId !== "string" ||
    (destinationColumnData.subGroupId !== undefined && typeof destinationColumnData.subGroupId !== "string") ||
    (destinationIssueData && (typeof destinationIssueData.id !== "string" || !destinationIssueData.id))
  )
    return;

  return {
    groupId: destinationColumnData.groupId,
    subGroupId: destinationColumnData.subGroupId as string | undefined,
    columnId: `${destinationColumnData.groupId}__${destinationColumnData.subGroupId}`,
    id: destinationIssueData?.id as string | undefined,
    canAddIssueBelow: !!destinationIssueData && extractClosestEdge(destinationIssueData) === "bottom",
  };
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

export const handleGroupDragDrop = async (
  source: GroupDropLocation,
  destination: GroupDropLocation,
  getIssueById: (id: string) => Issue | undefined,
  getIssueIds: (group?: string, subgroup?: string) => string[] | undefined,
  updateIssueOnDrop: (
    project: string,
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop: IssueDrop
  ) => Promise<void>,
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy: TIssueGroupByOptions | undefined,
  shouldAddIssueAtTop = false
) => {
  if (!source.id || source.id === destination.id) return false;
  const issue = getIssueById(source.id);
  if (!issue) return false;
  const ids = (getIssueIds(destination.groupId, destination.subGroupId) ?? []).filter((id) => id !== source.id);
  let index = destination.id ? ids.indexOf(destination.id) : shouldAddIssueAtTop ? 0 : ids.length;
  // An invalid captured card is not the same as an intentional column-tail drop.
  if (index < 0) return false;
  if (destination.id && destination.canAddIssueBelow) index++;
  const drop: IssueDrop = {
    group: destination.groupId,
    subgroup: destination.subGroupId,
    beforeId: ids[index],
    afterId: ids[index - 1],
  };
  const makeBody = (held: Issue): IssueUpdate => {
    const body: IssueUpdate = {};
    const move = (field: TIssueGroupByOptions | undefined, from: string | undefined, to: string | undefined) => {
      if (!field || to === undefined || from === to) return;
      switch (field) {
        case "priority":
          if (!["none", "low", "medium", "high", "urgent"].includes(to)) throw new Error("Invalid priority drop");
          body.priority = to as IssueUpdate["priority"];
          break;
        case "state":
          if (!to || to === "none") throw new Error("Invalid state drop");
          body.state_id = to;
          break;
        case "assignees":
        case "labels": {
          const key = field === "assignees" ? "assignee_ids" : "label_ids";
          body[key] = to === "none" ? [] : [...new Set([...held[key].filter((id) => id !== from), to])];
          break;
        }
        default:
          throw new Error("Unsupported drop group");
      }
    };
    move(groupBy, source.groupId, destination.groupId);
    move(subGroupBy, source.subGroupId, destination.subGroupId);
    return body;
  };
  await updateIssueOnDrop(issue.project_id, issue.id, makeBody, drop);
  return true;
};

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
 * This method returns the icon for Spreadsheet column headers
 * @param iconKey
 */
export function SpreadSheetPropertyIcon(props: ISvgIcons & { iconKey: string }) {
  const { iconKey, ...iconProps } = props;
  const Icon = SpreadSheetPropertyIconMap[iconKey];
  if (!Icon) return null;
  return <Icon {...iconProps} />;
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

const SpreadSheetPropertyIconMap: Record<string, FC<ISvgIcons>> = {
  MembersOutline: MembersOutline,
  CalendarOutline: CalendarOutline,
  DueDateOutline: DueDateOutline,
  LabelsOutline: LabelsOutline,
  PriorityOutline: PriorityOutline,
  StartDateOutline: StartDateOutline,
  StateOutline: StateOutline,
  Link2: LinkOutline,
  WorkItemsOutline: WorkItemsOutline,
};

export const SPREADSHEET_COLUMNS: { [key in keyof IIssueDisplayProperties]: TSpreadsheetColumn } = {
  assignee: SpreadsheetAssigneeColumn,
  created_on: SpreadsheetCreatedOnColumn,
  due_date: SpreadsheetDueDateColumn,
  labels: SpreadsheetLabelColumn,
  link: SpreadsheetLinkColumn,
  priority: SpreadsheetPriorityColumn,
  start_date: SpreadsheetStartDateColumn,
  state: SpreadsheetStateColumn,
  sub_issue_count: SpreadsheetSubIssueColumn,
  updated_on: SpreadsheetUpdatedOnColumn,
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
