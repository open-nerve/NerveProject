/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { StateGroup } from "@nerve/api-client";

export type TDraggableData = {
  groupKey: StateGroup;
  id: string;
};

export const STATE_GROUPS: {
  [key in StateGroup]: {
    key: StateGroup;
    label: string;
    /** The key of the group's name in the page's language. */
    i18n_label: string;
    defaultStateName: string;
    color: string;
  };
} = {
  backlog: {
    key: "backlog",
    label: "Backlog",
    i18n_label: "project_settings.states.groups.backlog",
    defaultStateName: "Backlog",
    color: "#d9d9d9",
  },
  unstarted: {
    key: "unstarted",
    label: "Unstarted",
    i18n_label: "project_settings.states.groups.unstarted",
    defaultStateName: "Todo",
    color: "#3f76ff",
  },
  started: {
    key: "started",
    label: "Started",
    i18n_label: "project_settings.states.groups.started",
    defaultStateName: "In Progress",
    color: "#f59e0b",
  },
  completed: {
    key: "completed",
    label: "Completed",
    i18n_label: "project_settings.states.groups.completed",
    defaultStateName: "Done",
    color: "#16a34a",
  },
  cancelled: {
    key: "cancelled",
    label: "Canceled",
    i18n_label: "project_settings.states.groups.cancelled",
    defaultStateName: "Cancelled",
    color: "#dc2626",
  },
};

export const ARCHIVABLE_STATE_GROUPS = [STATE_GROUPS.completed.key, STATE_GROUPS.cancelled.key];

export const PROGRESS_STATE_GROUPS_DETAILS = [
  {
    key: "completed_issues",
    title: "Completed",
    color: "#16A34A",
  },
  {
    key: "started_issues",
    title: "Started",
    color: "#F59E0B",
  },
  {
    key: "unstarted_issues",
    title: "Unstarted",
    color: "#3A3A3A",
  },
  {
    key: "backlog_issues",
    title: "Backlog",
    color: "#A3A3A3",
  },
];
