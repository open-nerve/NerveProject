/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import {
  AttachOutline,
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
import type { IIssueDisplayProperties, TSpreadsheetColumn } from "@nerve/types";

// components
import {
  SpreadsheetAssigneeColumn,
  SpreadsheetAttachmentColumn,
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

const SpreadSheetPropertyIconMap: Record<string, FC<ISvgIcons>> = {
  MembersOutline: MembersOutline,
  CalendarOutline: CalendarOutline,
  DueDateOutline: DueDateOutline,
  LabelsOutline: LabelsOutline,
  PriorityOutline: PriorityOutline,
  StartDateOutline: StartDateOutline,
  StateOutline: StateOutline,
  Link2: LinkOutline,
  AttachOutline: AttachOutline,
  WorkItemsOutline: WorkItemsOutline,
};

export const SPREADSHEET_COLUMNS: { [key in keyof IIssueDisplayProperties]: TSpreadsheetColumn } = {
  assignee: SpreadsheetAssigneeColumn,
  attachment_count: SpreadsheetAttachmentColumn,
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
