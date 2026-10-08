/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EUserPermissions } from "@nerve/types";

const paramsToKey = (params: any) => {
  const {
    state,
    state_group,
    priority,
    mentions,
    assignees,
    created_by,
    labels,
    start_date,
    target_date,
    sub_issue,
    project,
    layout,
    subscriber,
  } = params;

  let projectKey = project ? project.split(",") : [];
  let stateKey = state ? state.split(",") : [];
  let stateGroupKey = state_group ? state_group.split(",") : [];
  let priorityKey = priority ? priority.split(",") : [];
  let mentionsKey = mentions ? mentions.split(",") : [];
  let assigneesKey = assignees ? assignees.split(",") : [];
  let createdByKey = created_by ? created_by.split(",") : [];
  let labelsKey = labels ? labels.split(",") : [];
  let subscriberKey = subscriber ? subscriber.split(",") : [];
  const startDateKey = start_date ?? "";
  const targetDateKey = target_date ?? "";
  const type = params.type ? params.type.toUpperCase() : "NULL";
  const groupBy = params.group_by ? params.group_by.toUpperCase() : "NULL";
  const orderBy = params.order_by ? params.order_by.toUpperCase() : "NULL";
  const layoutKey = layout ? layout.toUpperCase() : "";

  // sorting each keys in ascending order
  projectKey = projectKey.toSorted().join("_");
  stateKey = stateKey.toSorted().join("_");
  stateGroupKey = stateGroupKey.toSorted().join("_");
  priorityKey = priorityKey.toSorted().join("_");
  assigneesKey = assigneesKey.toSorted().join("_");
  mentionsKey = mentionsKey.toSorted().join("_");
  createdByKey = createdByKey.toSorted().join("_");
  labelsKey = labelsKey.toSorted().join("_");
  subscriberKey = subscriberKey.toSorted().join("_");

  return `${layoutKey}_${projectKey}_${stateGroupKey}_${stateKey}_${priorityKey}_${assigneesKey}_${mentionsKey}_${createdByKey}_${type}_${groupBy}_${orderBy}_${labelsKey}_${startDateKey}_${targetDateKey}_${sub_issue}_${subscriberKey}`;
};

export const WORKSPACE_MODULES = (workspaceSlug: string) => `WORKSPACE_MODULES_${workspaceSlug.toUpperCase()}`;

export const WORKSPACE_CYCLES = (workspaceSlug: string) => `WORKSPACE_CYCLES_${workspaceSlug.toUpperCase()}`;

export const WORKSPACE_LABELS = (workspaceSlug: string) => `WORKSPACE_LABELS_${workspaceSlug.toUpperCase()}`;

export const WORKSPACE_PROJECTS_ROLES_INFORMATION = (workspaceSlug: string) =>
  `WORKSPACE_PROJECTS_ROLES_INFORMATION_${workspaceSlug.toUpperCase()}`;

export const WORKSPACE_STATES = (workspaceSlug: string) => `WORKSPACE_STATES_${workspaceSlug.toUpperCase()}`;

// cycles
export const CYCLE_ISSUES_WITH_PARAMS = (cycleId: string, params?: any) => {
  if (!params) return `CYCLE_ISSUES_WITH_PARAMS_${cycleId.toUpperCase()}`;

  const paramsKey = paramsToKey(params);

  return `CYCLE_ISSUES_WITH_PARAMS_${cycleId.toUpperCase()}_${paramsKey.toUpperCase()}`;
};

// Issues
export const ISSUE_DETAILS = (issueId: string) => `ISSUE_DETAILS_${issueId.toUpperCase()}`;

// project level keys
export const PROJECT_DETAILS = (_workspaceSlug: string, projectId: string) =>
  `PROJECT_DETAILS_${projectId.toUpperCase()}`;

export const PROJECT_ME_INFORMATION = (_workspaceSlug: string, projectId: string) =>
  `PROJECT_ME_INFORMATION_${projectId.toUpperCase()}`;

export const PROJECT_LABELS = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_LABELS_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_MEMBERS = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_MEMBERS_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_STATES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_STATES_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_MEMBER_PREFERENCES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_MEMBER_PREFERENCES_${projectId.toUpperCase()}_${projectRole}`;
