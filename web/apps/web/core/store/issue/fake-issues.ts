/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type {
  Issue,
  IssueDetail,
  IssueListPage,
  IssueLink,
  IssueComment,
  IssueReaction,
  IssueActivity,
} from "@nerve/api-client";
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

export const issueLinkOf = (id: string, issueId = "x"): IssueLink => ({
  id,
  issue_id: issueId,
  title: null,
  url: "https://example.test/",
  metadata: {},
  created_by_id: null,
  created_at: "2026-10-30T00:00:00Z",
  updated_at: "2026-10-30T00:00:00Z",
});

export const issueCommentOf = (id: string, issueId = "x"): IssueComment => ({
  id,
  issue_id: issueId,
  actor_id: "actor",
  comment_html: `<p>${id}</p>`,
  edited_at: null,
  created_at: "2026-10-30T00:00:00.123456Z",
  updated_at: "2026-10-30T00:00:00.123456Z",
  reactions: [],
});

export const issueReactionOf = (id: string, issueId = "x", actorId = "u-actor"): IssueReaction => ({
  id,
  issue_id: issueId,
  actor_id: actorId,
  reaction: "128077",
  created_at: "2026-10-30T00:00:00.123456Z",
});

/** Start an operation and hold its one real HTTP request until the test answers it. */
export async function requestedIssue<T>(nerve: FakeNerve, request: () => Promise<T>, endpoint: Endpoint) {
  const at = nerve.calls.length;
  const promise = request();
  const result = track(promise);
  await until(() => nerve.calls.length > at || result.settled, "issue request");
  expect(result.settled).toBe(false);
  expect(nerve.calls).toHaveLength(at + 1);
  const call = nerve.calls[at]!;
  expect([call.method, call.path]).toEqual(endpoint);
  return { result, call, promise };
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
let pageSource = 0;
export const pageMeta = (
  part: import("@/lib/paged-groups").PagePart | null = null,
  lower: import("@/lib/paged-groups").Boundary | null = null
): import("@/lib/paged-groups").PageReadMeta => ({
  sourceID: ++pageSource,
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

/** Real session stores and directory responses shared by issue HTTP behavior tests. */
export async function issueStoreTab() {
  const [{ RootStore }, { FakeNerve }, { workspaceOf, loadWorkspaces }, { projectOf, loadProjects }, { runInAction }] =
    await Promise.all([
      import("@/store/root.store"),
      import("@/lib/auth/fake-nerve"),
      import("@/store/workspace/fake-workspaces"),
      import("@/store/project/fake-projects"),
      import("mobx"),
    ]);
  const nerve = new FakeNerve();
  const api = nerve.client();
  const app = new RootStore(api);
  const workspace = workspaceOf("ws");
  const project = projectOf("A", workspace.id, { id: "A" });
  runInAction(() => {
    app.router.setQuery({ workspaceSlug: workspace.slug, projectId: project.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace]);
  await loadProjects(nerve, app.projectRoot.project, workspace, [project]);
  nerve.calls.length = 0;
  return { app, nerve, api, workspace, project };
}

export function issueActivityOf(id: string, issueId = "x", projectId = "A"): IssueActivity {
  return {
    id,
    issue_id: issueId,
    project_id: projectId,
    actor_id: "u-actor",
    field: "name",
    verb: "updated",
    old_value: "before",
    new_value: "after",
    old_identifier: null,
    new_identifier: null,
    created_at: "2026-10-30T00:00:00.123456789Z",
  };
}
