/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueDetail, IssueListPage } from "@nerve/api-client";
import { expect } from "vitest";
import type { Group } from "@/lib/paged-groups";
import type { Endpoint, FakeNerve } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";

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

/** A detail answer includes HTML; a list row deliberately does not. */
export const issueDetailOf = (id: string, projectId = "A"): IssueDetail => ({
  ...issueOf(id, projectId),
  description_html: "<p>Server detail</p>",
});

/** A flat server page; opaque continuation tokens are copied, never interpreted. */
export const issuePageOf = (data: Issue[], nextCursor: string | null = null): IssueListPage => ({
  total: data.length,
  data,
  next_cursor: nextCursor,
});

/** Start an operation and hold its one real HTTP request until the test answers it. */
export async function requestedIssue<T>(nerve: FakeNerve, request: () => Promise<T>, endpoint: Endpoint) {
  const at = nerve.calls.length;
  const result = track(request());
  await until(() => nerve.calls.length > at || result.settled, "issue request");
  expect(result.settled).toBe(false);
  expect(nerve.calls).toHaveLength(at + 1);
  const call = nerve.calls[at]!;
  expect([call.method, call.path]).toEqual(endpoint);
  return { result, call };
}

/** Shared pure list fixtures; no transport or second entity cache. */
export const listContext = (
  query: import("@/services/issue/issue-list.service").ProjectIssueQuery = {}
): import("@/lib/paged-groups").ListContext => ({
  scope: "project",
  projectID: "A",
  query: { order_by: "sort_order", ...query },
  currentUserID: "actor",
  catalog: { states: new Map(), labels: new Map(), members: new Map() },
  pendingIDs: new Set(),
});
export const pageMeta = (
  part: import("@/lib/paged-groups").PagePart | null = null,
  lower: import("@/lib/paged-groups").Boundary | null = null
): import("@/lib/paged-groups").PageReadMeta => ({
  part,
  first: lower === null,
  resetCounters: [],
  rowParts: [],
  ranges: new Map(
    part && lower ? [[JSON.stringify([part.group, part.subgroup]), { lower, upper: null, complete: false }]] : []
  ),
});
export const listIssue = (id: string, key: number, changes: Partial<Issue> = {}): Issue => ({
  ...issueOf(id),
  sort_order: key,
  ...changes,
});
export const listRow = (
  id: string,
  key: number,
  updatedAt = "2026-10-28T00:00:00Z"
): import("@/lib/paged-groups").Row => ({ id, key, updatedAt });
export const listGroup = (
  total: number,
  rows: import("@/lib/paged-groups").Row[] = [],
  complete = false,
  edge = 10
): Group => ({
  total,
  rows,
  nextCursor: complete ? null : "opaque-next",
  boundary: { id: "edge", key: edge },
  cursorBoundary: { id: "edge", key: edge },
  stale: false,
});
