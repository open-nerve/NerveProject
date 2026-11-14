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
  TIssueOrderByOptions,
} from "./view-props";

export type TIssuePriorities = "urgent" | "high" | "medium" | "low" | "none";

export type GroupByColumnTypes =
  | "project"
  | "state"
  | "state_detail.group"
  | "priority"
  | "labels"
  | "assignees"
  | "created_by"
  | "cycle"
  | "module";

/**
 * nerve's problem code refusing a work item's joining a group's cycle or module (P7b spec 2.4, R4): an ended cycle, an
 * archived cycle or module, and either of a project that has the domain turned off.
 */
export type GroupJoinRefusal =
  | "planning.cycle_completed"
  | "planning.archived"
  | "planning.cycles_disabled"
  | "planning.modules_disabled";

export interface IGroupByColumn {
  id: string;
  name: string;
  icon?: ReactElement | undefined;
  payload: Partial<IssueDetail>;
  /** The group of an archived cycle or module: its header says so after its name (P7b R4). */
  archived?: boolean;
  /**
   * Why nerve refuses a work item's joining the group, by its problem code (P7b R4): the one field a drop on it, its
   * new work items and its "add existing" all read.
   */
  refusal?: GroupJoinRefusal;
}

export interface ILayoutDisplayFiltersOptions {
  display_properties: (keyof IIssueDisplayProperties)[];
  display_filters: {
    group_by?: TIssueGroupByOptions[];
    sub_group_by?: TIssueGroupByOptions[];
    order_by?: TIssueOrderByOptions[];
  };
  extra_options: {
    access: boolean;
    values: TIssueExtraOptions[];
  };
}
