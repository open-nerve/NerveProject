/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export enum EIssueLayoutTypes {
  LIST = "list",
  KANBAN = "kanban",
  CALENDAR = "calendar",
  SPREADSHEET = "spreadsheet",
}

export enum EIssuesStoreType {
  PROFILE = "PROFILE",
  PROJECT = "PROJECT",
  CYCLE = "CYCLE",
  MODULE = "MODULE",
  ARCHIVED = "ARCHIVED",
  DEFAULT = "DEFAULT",
  WORKSPACE_DRAFT = "WORKSPACE_DRAFT",
}

export type TWorkItemWidgets = "sub-work-items" | "relations" | "links" | "attachments";

export interface IWorkItemPeekOverview {
  embedIssue?: boolean;
  embedRemoveCurrentNotification?: () => void;
}
