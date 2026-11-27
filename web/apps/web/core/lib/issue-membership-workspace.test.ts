/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Whether a row is in a workspace's list goes by the list's own words (M7/P6b spec 2.3, re-97 I1): the workspace, an
// open work item and its projects, as a member's list; its state, assignees and creator, `me` the caller, as a project's
// list; a mention or a subscription condition leaves it unknown, to be read again; no relation. A default view's own
// term is one of those words ("Assigned to me" is assignee_id: ["me"]), so an inline edit or a creation lists right at
// once, not only at the next read.
import { expect, test } from "vitest";
import type { WorkspaceIssueQuery } from "@/services/issue/issue-list.service";
import { listGroup, listIssue, listRow, workspaceListContext } from "@/store/issue/fake-issues";
import { issueMembership, issueParts } from "./issue-membership";
import { confirmGroups as confirmWithSources, type PagedGroups } from "./paged-groups";

// Hand-built fixtures stand for source 0, held before the change was sent (paged-groups.changes.test.ts).
const confirmGroups = (...args: Parameters<typeof confirmWithSources>) =>
  confirmWithSources(args[0], args[1], args[2], new Set([0]));
const afterTime = "2026-10-28T00:00:00.000001Z";
/** A whole flat list of five, every page read: x is listed at key 4. */
const listed = (): PagedGroups => ({ total: 5, groups: new Map([["", listGroup(5, [listRow("x", 4)], true)]]) });

test("on 'Assigned to me' a row whose assignee becomes another leaves once nerve confirms it, the total one less", () => {
  const context = workspaceListContext({ assignee_id: ["me"] });
  const before = listIssue("x", 4, { assignee_ids: ["actor"] });
  const after = { ...before, assignee_ids: ["other"], updated_at: afterTime };
  expect(issueMembership(before, context)).toBe(true);
  expect(issueMembership(after, context)).toBe(false);
  const result = confirmGroups(listed(), { id: "x", beforeSnapshot: { kind: "known", issue: before }, after }, context);
  expect(result.value.total).toBe(4);
  expect(result.value.groups.get("")).toMatchObject({ total: 4, rows: [], stale: false });
});

test("under a state_id condition a row moved to another state leaves once nerve confirms it", () => {
  const context = workspaceListContext({ state_id: ["S1"] });
  const before = listIssue("x", 4, { state_id: "S1" });
  const after = { ...before, state_id: "S2", updated_at: afterTime };
  const result = confirmGroups(listed(), { id: "x", beforeSnapshot: { kind: "known", issue: before }, after }, context);
  expect(result.value.total).toBe(4);
  expect(result.value.groups.get("")).toMatchObject({ total: 4, rows: [] });
});

test("on 'Created by me' a row is the caller's creation, `me` standing for him", () => {
  const context = workspaceListContext({ created_by_id: ["me"] });
  expect(issueMembership(listIssue("mine", 1, { created_by_id: "actor" }), context)).toBe(true);
  expect(issueMembership(listIssue("theirs", 1, { created_by_id: "other" }), context)).toBe(false);
});

test("on 'Assigned to me' a new work item assigned to another is not put in the list", () => {
  const context = workspaceListContext({ assignee_id: ["me"] });
  const value = listed();
  const created = listIssue("y", 3, { assignee_ids: ["other"] });
  const result = confirmGroups(value, { id: "y", beforeSnapshot: { kind: "absent" }, after: created }, context);
  expect(result.value).toMatchObject({ total: 5, groups: new Map([["", { ...value.groups.get("")!, total: 5 }]]) });
  const mine = listIssue("z", 3, { assignee_ids: ["actor"] });
  const joined = confirmGroups(value, { id: "z", beforeSnapshot: { kind: "absent" }, after: mine }, context);
  expect(joined.value.total).toBe(6);
  expect(joined.value.groups.get("")?.rows).toEqual([listRow("z", 3), listRow("x", 4)]);
});

test.each<[string, WorkspaceIssueQuery]>([
  ["subscriber_id", { subscriber_id: ["me"] }],
  ["mention_id", { mention_id: ["me"] }],
])("under a %s condition a confirmation is read again, never put in the list", (_word, query) => {
  const context = workspaceListContext(query);
  const created = listIssue("y", 3);
  expect(issueMembership(created, context)).toBeUndefined();
  const result = confirmGroups(listed(), { id: "y", beforeSnapshot: { kind: "absent" }, after: created }, context);
  expect(result.value.total).toBe(5);
  expect(result.value.groups.get("")).toMatchObject({ total: 5, rows: [listRow("x", 4)], stale: true });
});

test("a workspace list keeps to its workspace, its projects and open work items, and has no relation", () => {
  const context = workspaceListContext({ project_id: ["A"] });
  expect(issueMembership(listIssue("x", 1), context)).toBe(true);
  expect(issueMembership(listIssue("x", 1, { project_id: "B" }), context)).toBe(false);
  expect(issueMembership(listIssue("x", 1, { workspace_id: "other-ws" }), context)).toBe(false);
  expect(issueMembership(listIssue("x", 1, { archived_at: "2026-10-28T00:00:00Z" }), context)).toBe(false);
  // nobody's relation: the member list's assignee check does not apply
  expect(issueMembership(listIssue("x", 1, { assignee_ids: [] }), workspaceListContext())).toBe(true);
});

test("a confirmation in another workspace leaves the list as it is, its earlier snapshot unknown or not", () => {
  const context = workspaceListContext({ assignee_id: ["me"] });
  const value = listed();
  const foreign = listIssue("x", 4, { workspace_id: "other-ws", assignee_ids: ["actor"], updated_at: afterTime });
  for (const beforeSnapshot of [{ kind: "unknown" }, { kind: "absent" }] as const)
    expect(confirmGroups(value, { id: "x", beforeSnapshot, after: foreign }, context)).toMatchObject({
      value,
      verifyIds: [],
    });
});

test("a workspace list is flat: its query's grouping reads none, a row in its one group", () => {
  const context = workspaceListContext({ assignee_id: ["me"] });
  expect(context.query.group_by).toBeUndefined();
  expect(issueParts(listIssue("x", 1, { assignee_ids: ["actor"] }), context)).toEqual([{ group: "" }]);
  expect(issueMembership(listIssue("x", 1, { assignee_ids: ["actor"] }), context, { kind: "group", group: "" })).toBe(
    true
  );
});
