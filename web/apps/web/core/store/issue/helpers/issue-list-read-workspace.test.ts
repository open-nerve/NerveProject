/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's list is a scope of its own (M7/P6b spec 2.3, R1): every role of the workspace reads it, its guests too
// (the rule of workspace_issue.list), where a member's list and a project's archive take an admin or a member; it reads
// nerve's workspace list, flat, a later page by its cursor alone; it takes a page only of its workspace and, when it
// names projects, of them; a write in a project it covers comes into it; and it does not hang on the caller's
// membership of the workspace, as a member's list does. The stores and HTTP are real.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { IssueListBinding } from "./issue-list-binding";
import { allowsRole, type IssueListScope } from "./issue-list-read";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

const at = { workspaceSlug: "ws", workspaceID: "id-ws" };
const workspaceList = (query: Extract<IssueListScope, { scope: "workspace" }>["query"] = {}) =>
  new IssueListBinding(tab.app.issue, { ...at, scope: "workspace", query: { order_by: "-created_at", ...query } });
const LIST: [string, string] = ["GET", "/api/v0/workspaces/ws/issues"];
/** A work item of project, created at the hour given: the newest first is the lists' order here. */
const createdAt = (id: string, hour: number, project = "A") => ({
  ...issueOf(id, project),
  created_at: `2026-10-28T0${hour}:00:00Z`,
});

it.each([
  [undefined, [false, false, false, false]],
  [5, [true, false, false, true]],
  [15, [true, true, true, true]],
  [20, [true, true, true, true]],
])("with the role %s reads a project's open list, its archive, a member's list, a workspace's: %j", (role, read) => {
  const scopes: IssueListScope[] = [
    { ...at, scope: "project", projectID: "A", query: {} },
    { ...at, scope: "project", projectID: "A", query: { archived: "true" } },
    { ...at, scope: "member", userID: "actor", query: { relation: "assigned" } },
    { ...at, scope: "workspace", query: {} },
  ];
  expect(scopes.map((scope) => allowsRole(scope, role))).toEqual(read);
});

it("reads nerve's workspace list with its query, flat, and a later page by its cursor alone", async () => {
  const list = workspaceList({ assignee_id: ["me"], project_id: ["A", "B"], sub_issue: "true", limit: 2 });
  const first = await requestedIssue(tab.nerve, () => list.fetchFirst(), LIST);
  expect(first.call.query).toEqual({
    assignee_id: "me",
    project_id: "A,B",
    order_by: "-created_at",
    sub_issue: "true",
    limit: "2",
  });
  first.call.answer(json(200, { ...issuePageOf([createdAt("x", 3), createdAt("y", 2, "B")], "c1"), total: 3 }));
  await first.promise;
  expect(list.value?.total).toBe(3);
  expect(list.group("")?.rows.map((row) => row.id)).toEqual(["x", "y"]);
  expect(tab.app.issue.issues.held("y")?.project_id).toBe("B");
  const next = await requestedIssue(tab.nerve, () => list.fetchNext(""), LIST);
  expect(next.call.query).toEqual({ ...first.call.query, cursor: "c1" });
  next.call.answer(json(200, { ...issuePageOf([createdAt("z", 1)]), total: 3 }));
  await next.promise;
  expect(list.group("")?.rows.map((row) => row.id)).toEqual(["x", "y", "z"]);
});

it.each([
  ["a project its query leaves out", issueOf("y", "B")],
  ["another workspace", issueOf("y", "A", "other-ws")],
])("takes no row of %s", async (_where, row) => {
  const list = workspaceList({ project_id: ["A"] });
  const read = await requestedIssue(tab.nerve, () => list.fetchFirst(), LIST);
  read.call.answer(json(200, issuePageOf([issueOf("x"), row])));
  await expect(read.promise).rejects.toThrow("Issue page does not match its scope");
  expect(list.value).toBeUndefined();
  expect(tab.app.issue.issues.held("y")).toBeUndefined();
});

it("is read by the workspace's guest, as by its admins and members", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspaceOf("ws", { role: 5 })]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    projectOf("A", tab.workspace.id, { id: "A", member_role: 5 }),
  ]);
  const list = workspaceList();
  const read = await requestedIssue(tab.nerve, () => list.fetchFirst(), LIST);
  read.call.answer(json(200, issuePageOf([issueOf("x")])));
  await read.promise;
  expect(list.group("")?.rows.map((row) => row.id)).toEqual(["x"]);
});

/** A write in a project of a workspace, as the write's queue gives it to the lists. */
const write = (projectID: string, workspaceID = "id-ws") => ({ workspaceSlug: "ws", workspaceID, projectID });

it("covers a write in its workspace's projects, or in those its query names", () => {
  const whole = workspaceList();
  const named = workspaceList({ project_id: ["A"] });
  expect([whole.covers(write("A")), whole.covers(write("B")), whole.covers(write("A", "other-ws"))]).toEqual([
    true,
    true,
    false,
  ]);
  expect([named.covers(write("A")), named.covers(write("B"))]).toEqual([true, false]);
});

it("keeps its rows when the caller's membership of the workspace changes, where a member's list is read again", async () => {
  const workspace = workspaceList();
  const member = new IssueListBinding(tab.app.issue, {
    ...at,
    scope: "member",
    userID: "actor",
    query: { relation: "created", order_by: "-created_at" },
  });
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => call.answer(json(200, issuePageOf([issueOf("x")])));
  await Promise.all([workspace.fetchFirst(), member.fetchFirst()]);
  tab.nerve.respond = undefined;
  expect(tab.nerve.calls.map((call) => call.path)).toEqual([LIST[1], "/api/v0/workspaces/ws/members/actor/issues"]);
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspaceOf("ws", { role: 5 })]);
  expect(workspace.value?.total).toBe(1);
  expect(member.value).toBeUndefined();
});
