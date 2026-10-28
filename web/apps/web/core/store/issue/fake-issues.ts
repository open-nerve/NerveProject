/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";

/** A complete service response fixture, never used to fill a production create request. */
export const issueOf = (id: string, projectId = "A", workspaceId = "id-ws"): Issue => ({
  id,
  project_id: projectId,
  workspace_id: workspaceId,
  sequence_id: 1,
  name: id,
  sort_order: 1000,
  state_id: "state",
  priority: "none",
  label_ids: [],
  assignee_ids: [],
  sub_issues_count: 0,
  link_count: 0,
  parent_id: null,
  created_at: "2026-10-28T00:00:00Z",
  updated_at: "2026-10-28T00:00:00Z",
  start_date: null,
  target_date: null,
  completed_at: null,
  archived_at: null,
  created_by_id: "actor",
  updated_by_id: "actor",
});
