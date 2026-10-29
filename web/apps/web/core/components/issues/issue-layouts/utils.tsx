// oxlint-disable no-shadow
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueUpdate } from "@nerve/api-client";

import { extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";
import { clone, isNil } from "lodash-es";
import scrollIntoView from "smooth-scroll-into-view-if-needed";
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
import { Avatar } from "@makeplane/propel/components/avatar";
import { EIconSize, ISSUE_PRIORITIES, STATE_GROUPS } from "@nerve/constants";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import type { ISvgIcons } from "@nerve/propel/icons";
import { PriorityIcon, StateGroupIcon } from "@nerve/propel/icons";
import type {
  GroupByColumnTypes,
  IGroupByColumn,
  IIssueDisplayProperties,
  IPragmaticDropPayload,
  TIssueGroupByOptions,
  IIssueFilterOptions,
  IIssueFilters,
  IIssueDisplayFilterOptions,
  TGetColumns,
  TSpreadsheetColumn,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";

import { getFileURL } from "@nerve/utils";
// store
import { rootStore } from "@/lib/store-context";
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
  // oxlint-disable-next-line no-unneeded-ternary
  [EIssuesStoreType.PROFILE, EIssuesStoreType.WORKSPACE_DRAFT].includes(type) ? true : false;

type TGetGroupByColumns = {
  groupBy: GroupByColumnTypes | null;
  includeNone: boolean;
  isWorkspaceLevel: boolean;
  projectId?: string;
};

// NOTE: Type of groupBy is different compared to what's being passed from the components.
// We are using `as` to typecast it to the expected type.
// It can break the includeNone logic if not handled properly.
export const getGroupByColumns = ({
  groupBy,
  includeNone,
  isWorkspaceLevel,
  projectId,
}: TGetGroupByColumns): IGroupByColumn[] | undefined => {
  // If no groupBy is specified and includeNone is true, return "All Issues" group
  if (!groupBy && includeNone) {
    return [
      {
        id: "",
        name: "All work items",
        payload: {},
        icon: undefined,
      },
    ];
  }

  // Return undefined if no valid groupBy
  if (!groupBy) return undefined;

  // Map of group by options to their corresponding column getter functions
  const groupByColumnMap: Record<
    GroupByColumnTypes,
    ({ isWorkspaceLevel, projectId }: TGetColumns) => IGroupByColumn[] | undefined
  > = {
    project: getProjectColumns,
    state: getStateColumns,
    "state_detail.group": getStateGroupColumns,
    priority: getPriorityColumns,
    labels: getLabelsColumns,
    assignees: getAssigneeColumns,
    created_by: getCreatedByColumns,
  };

  // Get and return the columns for the specified group by option
  return groupByColumnMap[groupBy]?.({ isWorkspaceLevel, projectId });
};

const getProjectColumns = (): IGroupByColumn[] | undefined => {
  const { joinedProjectIds: projectIds, getProjectById } = rootStore.projectRoot.project;
  // Return undefined if no project ids
  if (!projectIds) return;
  // Map project ids to project columns
  return projectIds
    .map((projectId: string) => {
      const project = getProjectById(projectId);
      if (!project) return;
      return {
        id: project.id,
        name: project.name,
        icon: (
          <div className="grid h-6 w-6 flex-shrink-0 place-items-center">
            <Logo logo={project.logo_props} />
          </div>
        ),
        payload: { project_id: project.id },
      };
    })
    .filter((column) => column !== undefined) as IGroupByColumn[];
};

const getStateColumns = ({ projectId }: TGetColumns): IGroupByColumn[] | undefined => {
  const { getProjectStates, projectStates, getStatePercentageInGroup } = rootStore.state;
  const _states = projectId ? getProjectStates(projectId) : projectStates;
  if (!_states) return;
  // map project states to group by columns
  return _states.map((state) => ({
    id: state.id,
    name: state.name,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon
          stateGroup={state.group}
          color={state.color}
          size={EIconSize.LG}
          percentage={getStatePercentageInGroup(state.id)}
        />
      </div>
    ),
    payload: { state_id: state.id },
  }));
};

const getStateGroupColumns = (): IGroupByColumn[] => {
  // map state groups to group by columns
  return Object.values(STATE_GROUPS).map((stateGroup) => ({
    id: stateGroup.key,
    name: stateGroup.label,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon stateGroup={stateGroup.key} size={EIconSize.LG} />
      </div>
    ),
    payload: {},
  }));
};

const getPriorityColumns = (): IGroupByColumn[] => {
  // map priorities to group by columns
  return ISSUE_PRIORITIES.map((priority) => ({
    id: priority.key,
    name: priority.title,
    icon: <PriorityIcon priority={priority?.key} />,
    payload: { priority: priority.key },
  }));
};

const getLabelsColumns = ({ isWorkspaceLevel }: TGetColumns): IGroupByColumn[] => {
  // the workspace's labels across its projects are M7's (M3 design 7.3): a workspace level groups none of them
  const labels = [
    ...(isWorkspaceLevel ? [] : (rootStore.label.projectLabels ?? [])),
    { id: "none", name: "None", color: "#666" },
  ];
  // map labels to group by columns
  return labels.map((label) => ({
    id: label.id,
    name: label.name,
    icon: (
      <div className="h-[12px] w-[12px] rounded-full" style={{ backgroundColor: label.color ? label.color : "#666" }} />
    ),
    payload: label?.id === "none" ? {} : { label_ids: [label.id] },
  }));
};

const getAssigneeColumns = ({ isWorkspaceLevel, projectId }: TGetColumns): IGroupByColumn[] | undefined => {
  // store values
  const { getUserDetails } = rootStore.memberRoot;
  // derived values
  const { memberIds, includeNone } = getScopeMemberIds({ isWorkspaceLevel, projectId });
  const assigneeColumns: IGroupByColumn[] = [];

  if (!memberIds) return [];

  memberIds.forEach((memberId) => {
    const member = getUserDetails(memberId);
    if (!member) return;
    assigneeColumns.push({
      id: memberId,
      name: member?.display_name || "",
      icon: (
        <Avatar
          alt={member?.display_name}
          fallback={member?.display_name?.[0]?.toUpperCase()}
          src={getFileURL(member?.avatar_url ?? "")}
          size="xs"
        />
      ),
      payload: { assignee_ids: [memberId] },
    });
  });
  if (includeNone) {
    assigneeColumns.push({ id: "none", name: "None", icon: <Avatar size="xs" />, payload: {} });
  }

  return assigneeColumns;
};

const getCreatedByColumns = (): IGroupByColumn[] | undefined => {
  const {
    project: { projectMemberIds },
    getUserDetails,
  } = rootStore.memberRoot;
  if (!projectMemberIds) return;
  // Map project member ids to group by created by columns
  return projectMemberIds.map((memberId) => {
    const member = getUserDetails(memberId);
    return {
      id: memberId,
      name: member?.display_name || "",
      icon: (
        <Avatar
          alt={member?.display_name}
          fallback={member?.display_name?.[0]?.toUpperCase()}
          src={getFileURL(member?.avatar_url ?? "")}
          size="xs"
        />
      ),
      payload: {},
    };
  });
};

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
 * This Method finds the DOM element with elementId, scrolls to it and highlights the issue block
 * @param elementId
 * @param shouldScrollIntoView
 */
export const highlightIssueOnDrop = (elementId: string | undefined, shouldScrollIntoView = true) => {
  setTimeout(async () => {
    const sourceElementId = elementId ?? "";
    const sourceElement = document.getElementById(sourceElementId);
    sourceElement?.classList?.add(HIGHLIGHT_CLASS);
    if (shouldScrollIntoView && sourceElement)
      await scrollIntoView(sourceElement, { behavior: "smooth", block: "center", duration: 1500 });
  }, 200);
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

  if (typeof sourceIssueData?.id !== "string" || !sourceIssueData.id || typeof sourceColumData?.groupId !== "string")
    return;

  return {
    groupId: sourceColumData.groupId as string,
    subGroupId: sourceColumData.subGroupId as string,
    columnId: sourceColumData.columnId as string,
    id: sourceIssueData.id as string,
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

  if (typeof destinationColumnData?.groupId !== "string") return;

  // extract instruction from destination issue
  const extractedInstruction = destinationIssueData ? extractInstruction(destinationIssueData)?.type : "";

  return {
    groupId: destinationColumnData.groupId as string,
    subGroupId: destinationColumnData.subGroupId as string,
    columnId: destinationColumnData.columnId as string,
    id: destinationIssueData?.id as string | undefined,
    canAddIssueBelow: extractedInstruction === "reorder-below",
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
  if (!source.id || source.id === destination.id) return;
  const issue = getIssueById(source.id);
  if (!issue) return;
  const ids = (getIssueIds(destination.groupId, destination.subGroupId) ?? []).filter((id) => id !== source.id);
  let index = destination.id ? ids.indexOf(destination.id) : shouldAddIssueAtTop ? 0 : ids.length;
  if (index < 0) index = ids.length;
  if (destination.canAddIssueBelow && index === ids.length - 1) index++;
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
  const { iconKey } = props;
  const Icon = SpreadSheetPropertyIconMap[iconKey];
  if (!Icon) return null;
  return <Icon {...props} />;
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

  const isDisplayFiltersApplied = Object.keys(filters.displayFilters ?? {}).some((key) => {
    const value = filters.displayFilters?.[key as keyof IIssueDisplayFilterOptions];
    if (!value) return false;
    // -create_at is the default order
    if (key === "order_by") {
      return value !== "-created_at";
    }
    return true;
  });

  return isDisplayPropertiesApplied || isDisplayFiltersApplied;
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
 * @param maxSequenceId - Maximum sequence ID in the project (e.g., 1234)
 * @returns Width in pixels needed to display the identifier
 *
 * @example
 * // For "PROJ-1234"
 * calculateIdentifierWidth(4, 1234) // Returns width for "PROJ" + "-" + "1234"
 */
export const calculateIdentifierWidth = (projectIdentifierLength: number, maxSequenceId: number): number => {
  const sequenceDigits = Math.max(1, Math.floor(Math.log10(maxSequenceId)) + 1);
  return projectIdentifierLength * 7 + 7 + sequenceDigits * 7; // project identifier chars + dash + sequence digits
};

type TGetScopeMemberIdsResult = {
  memberIds: string[];
  includeNone: boolean;
};

const getScopeMemberIds = ({ isWorkspaceLevel, projectId }: TGetColumns): TGetScopeMemberIdsResult => {
  // store values
  const { workspaceMemberIds } = rootStore.memberRoot.workspace;
  const { projectMemberIds } = rootStore.memberRoot.project;
  // derived values
  const memberIds = workspaceMemberIds;

  if (isWorkspaceLevel) {
    return { memberIds: memberIds ?? [], includeNone: true };
  }

  if (projectId || (projectMemberIds && projectMemberIds.length > 0)) {
    const { getProjectMemberIds } = rootStore.memberRoot.project;
    const _projectMemberIds = projectId ? getProjectMemberIds(projectId, false) : projectMemberIds;
    return {
      memberIds: _projectMemberIds ?? [],
      includeNone: true,
    };
  }

  return { memberIds: [], includeNone: true };
};

const SpreadSheetPropertyIconMap: Record<string, FC<ISvgIcons>> = {
  MembersOutline: MembersOutline,
  CalenderDays: CalendarOutline,
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
