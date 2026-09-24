/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { differenceInCalendarDays } from "date-fns/differenceInCalendarDays";
import { isEmpty } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, STATE_GROUPS } from "@nerve/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  TIssue,
  TIssueParams,
  TStateGroups,
} from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
// local imports
import { getDate } from "../datetime";
import { isEditorEmpty } from "../editor";

export const handleIssueQueryParamsByLayout = (
  layout: EIssueLayoutTypes | undefined,
  viewType: "my_issues" | "issues" | "profile_issues" | "archived_issues" | "draft_issues"
): TIssueParams[] | null => {
  const queryParams: TIssueParams[] = ["filters"];

  if (!layout) return null;

  const currentViewLayoutOptions = ISSUE_DISPLAY_FILTERS_BY_PAGE[viewType].layoutOptions[layout];

  // add display filters query params
  Object.keys(currentViewLayoutOptions.display_filters).forEach((option) => {
    queryParams.push(option as TIssueParams);
  });

  // add extra options query params
  if (currentViewLayoutOptions.extra_options.access) {
    currentViewLayoutOptions.extra_options.values.forEach((option) => {
      queryParams.push(option);
    });
  }

  return queryParams;
};

/**
 *
 * @description create a full issue payload with some default values. This function also parse the form field
 * like assignees, labels, etc. and add them to the payload
 * @param projectId project id to be added in the issue payload
 * @param formData partial issue data from the form. This will override the default values
 * @returns full issue payload with some default values
 */
export const createIssuePayload: (projectId: string, formData: Partial<TIssue>) => TIssue = (
  projectId: string,
  formData: Partial<TIssue>
) => {
  const payload: TIssue = {
    id: uuidv4(),
    project_id: projectId,
    priority: "none",
    label_ids: [],
    assignee_ids: [],
    sub_issues_count: 0,
    attachment_count: 0,
    link_count: 0,
    // tempId is used for optimistic updates. It is not a part of the API response.
    tempId: uuidv4(),
    // to be overridden by the form data
    ...formData,
  } as TIssue;

  return payload;
};

/**
 * @description check if the issue due date should be highlighted
 * @param date
 * @param stateGroup
 * @returns boolean
 */
export const shouldHighlightIssueDueDate = (
  date: string | Date | null,
  stateGroup: TStateGroups | undefined
): boolean => {
  if (!date || !stateGroup) return false;
  // if the issue is completed or cancelled, don't highlight the due date
  if ([STATE_GROUPS.completed.key, STATE_GROUPS.cancelled.key].includes(stateGroup)) return false;

  const parsedDate = getDate(date);
  if (!parsedDate) return false;

  const targetDateDistance = differenceInCalendarDays(parsedDate, new Date());

  // if the issue is overdue, highlight the due date
  return targetDateDistance <= 0;
};

export const formatTextList = (TextArray: string[]): string => {
  const count = TextArray.length;
  switch (count) {
    case 0:
      return "";
    case 1:
      return TextArray[0];
    case 2:
      return `${TextArray[0]} and ${TextArray[1]}`;
    case 3:
      return `${TextArray.slice(0, 2).join(", ")}, and ${TextArray[2]}`;
    case 4:
      return `${TextArray.slice(0, 3).join(", ")}, and ${TextArray[3]}`;
    default:
      return `${TextArray.slice(0, 3).join(", ")}, and +${count - 3} more`;
  }
};

export const getDescriptionPlaceholderI18n = (isFocused: boolean, description: string | undefined): string => {
  const isDescriptionEmpty = isEditorEmpty(description);
  if (!isDescriptionEmpty || isFocused) return "common.press_for_commands";
  else return "common.click_to_add_description";
};

/**
 * @description This method is used to apply the display filters on the issues
 * @param {IIssueDisplayFilterOptions} displayFilters
 * @returns {IIssueDisplayFilterOptions}
 */
export const getComputedDisplayFilters = (
  displayFilters: IIssueDisplayFilterOptions = {},
  defaultValues?: IIssueDisplayFilterOptions
): IIssueDisplayFilterOptions => {
  const filters = !isEmpty(displayFilters) ? displayFilters : defaultValues;
  return {
    calendar: {
      show_weekends: filters?.calendar?.show_weekends || false,
      layout: filters?.calendar?.layout || "month",
    },
    layout: filters?.layout || EIssueLayoutTypes.LIST,
    order_by: filters?.order_by || "sort_order",
    group_by: filters?.group_by || null,
    sub_group_by: filters?.sub_group_by || null,
    sub_issue: filters?.sub_issue || false,
    show_empty_groups: filters?.show_empty_groups || false,
  };
};

/**
 * @description This method is used to apply the display properties on the issues
 * @param {IIssueDisplayProperties} displayProperties
 * @returns {IIssueDisplayProperties}
 */
export const getComputedDisplayProperties = (
  displayProperties: IIssueDisplayProperties = {}
): IIssueDisplayProperties => ({
  assignee: displayProperties?.assignee ?? true,
  start_date: displayProperties?.start_date ?? true,
  due_date: displayProperties?.due_date ?? true,
  labels: displayProperties?.labels ?? true,
  priority: displayProperties?.priority ?? true,
  state: displayProperties?.state ?? true,
  sub_issue_count: displayProperties?.sub_issue_count ?? true,
  attachment_count: displayProperties?.attachment_count ?? true,
  link: displayProperties?.link ?? true,
  key: displayProperties?.key ?? true,
  created_on: displayProperties?.created_on ?? true,
  updated_on: displayProperties?.updated_on ?? true,
  modules: displayProperties?.modules ?? true,
  cycle: displayProperties?.cycle ?? true,
});

export const generateWorkItemLink = ({
  workspaceSlug,
  projectId,
  issueId,
  projectIdentifier,
  sequenceId,
  isArchived = false,
}: {
  workspaceSlug: string | undefined | null;
  projectId: string | undefined | null;
  issueId: string | undefined | null;
  projectIdentifier: string | undefined | null;
  sequenceId: string | number | undefined | null;
  isArchived?: boolean;
}): string => {
  const archiveIssueLink = `/${workspaceSlug}/projects/${projectId}/archives/issues/${issueId}`;
  const workItemLink = `/${workspaceSlug}/browse/${projectIdentifier}-${sequenceId}`;

  return isArchived ? archiveIssueLink : workItemLink;
};
