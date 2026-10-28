/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

export enum EIssueLayoutTypes {
  LIST = "list",
  KANBAN = "kanban",
  CALENDAR = "calendar",
  SPREADSHEET = "spreadsheet",
}

export enum EIssuesStoreType {
  PROFILE = "PROFILE",
  PROJECT = "PROJECT",
  ARCHIVED = "ARCHIVED",
  DEFAULT = "DEFAULT",
  WORKSPACE_DRAFT = "WORKSPACE_DRAFT",
}

type TIssueResponseResults =
  | Issue[]
  | {
      [key: string]: {
        results:
          | Issue[]
          | {
              [key: string]: {
                results: Issue[];
                total_results: number;
              };
            };
        total_results: number;
      };
    };

export type TIssuesResponse = {
  grouped_by: string;
  next_cursor: string;
  prev_cursor: string;
  next_page_results: boolean;
  prev_page_results: boolean;
  total_count: number;
  count: number;
  total_pages: number;
  extra_stats: null;
  results: TIssueResponseResults;
  total_results: number;
};

export type TWorkItemWidgets = "sub-work-items" | "relations" | "links" | "attachments";

export interface IWorkItemPeekOverview {
  embedIssue?: boolean;
  embedRemoveCurrentNotification?: () => void;
  storeType?: EIssuesStoreType;
}

/** Legacy update request context; skip_activity must never enter an issue entity. */
export type TLegacyIssueUpdateRequest = Partial<IssueDetail> & { skip_activity?: string };
