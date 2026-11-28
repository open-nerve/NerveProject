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
  /** A project's view's page: its project's list under the page's draft of the view (P6a spec 2.5). */
  PROJECT_VIEW = "PROJECT_VIEW",
  /** A workspace's view's page: the workspace's list under the page's preferences (M7/P6b spec 2.3). */
  GLOBAL = "GLOBAL",
  ARCHIVED = "ARCHIVED",
  DEFAULT = "DEFAULT",
  WORKSPACE_DRAFT = "WORKSPACE_DRAFT",
}

export type TWorkItemWidgets = "sub-work-items" | "relations" | "links" | "attachments";

export interface IWorkItemPeekOverview {
  embedIssue?: boolean;
  embedRemoveCurrentNotification?: () => void;
}
