/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import type { ReactElement } from "react";

import type {
  IIssueDisplayProperties,
  TIssueExtraOptions,
  TIssueGroupByOptions,
  TIssueGroupingFilters,
  TIssueOrderByOptions,
} from "./view-props";

export interface ILinkDetails {
  created_at: Date;
  created_by: string;
  id: string;
  metadata: any;
  title: string;
  url: string;
}

export type TIssuePriorities = "urgent" | "high" | "medium" | "low" | "none";

export type GroupByColumnTypes =
  | "project"
  | "state"
  | "state_detail.group"
  | "priority"
  | "labels"
  | "assignees"
  | "created_by";

export type TGetColumns = {
  isWorkspaceLevel?: boolean;
  projectId?: string;
};

export interface IGroupByColumn {
  id: string;
  name: string;
  icon?: ReactElement | undefined;
  payload: Partial<IssueDetail>;
  isDropDisabled?: boolean;
  dropErrorMessage?: string;
}

export interface ILayoutDisplayFiltersOptions {
  display_properties: (keyof IIssueDisplayProperties)[];
  display_filters: {
    group_by?: TIssueGroupByOptions[];
    sub_group_by?: TIssueGroupByOptions[];
    order_by?: TIssueOrderByOptions[];
    type?: TIssueGroupingFilters[];
  };
  extra_options: {
    access: boolean;
    values: TIssueExtraOptions[];
  };
}
