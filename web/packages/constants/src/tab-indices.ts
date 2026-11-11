/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

const INTAKE_ISSUE_CREATE_FORM_TAB_INDICES = [
  "name",
  "description_html",
  "priority",
  "assignee_ids",
  "label_ids",
  "start_date",
  "target_date",
  "cycle_id",
  "module_ids",
  "parent_id",
  "create_more",
  "discard_button",
  "submit_button",
];

const PROJECT_MODULE_TAB_INDICES = [
  "name",
  "description",
  "date_range",
  "status",
  "lead",
  "member_ids",
  "cancel",
  "submit",
];

const PROJECT_VIEW_TAB_INDICES = ["name", "description", "filters", "cancel", "submit"];

export enum ETabIndices {
  INTAKE_ISSUE_FORM = "intake-issue-form",
  PROJECT_MODULE = "project-module",
  PROJECT_VIEW = "project-view",
}

export const TAB_INDEX_MAP: Record<ETabIndices, string[]> = {
  [ETabIndices.INTAKE_ISSUE_FORM]: INTAKE_ISSUE_CREATE_FORM_TAB_INDICES,
  [ETabIndices.PROJECT_MODULE]: PROJECT_MODULE_TAB_INDICES,
  [ETabIndices.PROJECT_VIEW]: PROJECT_VIEW_TAB_INDICES,
};
