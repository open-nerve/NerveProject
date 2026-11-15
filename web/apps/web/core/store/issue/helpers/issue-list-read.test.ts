/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadProjects, projectOf, stateOf } from "@/store/project/fake-projects";
import { issuePageOf, issueStoreTab, listIssue, requestedIssue } from "../fake-issues";
import { IssueListBinding } from "./issue-list-binding";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

// A page waits for its directories, then checks its read again (M6 P7b spec 2.1): a scope that changed meanwhile
// accepts none of the page, not even the rows of a project that stayed readable.
it("a member page whose scope lost a project while its directory was read accepts none of its rows", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const member = new IssueListBinding(tab.app.issue, {
    scope: "member",
    workspaceSlug: "ws",
    workspaceID: tab.workspace.id,
    userID: "actor",
    query: { relation: "assigned", project_id: ["A", "B"], order_by: "state_name", limit: 10 },
  });
  const state = stateOf(tab.project, "Waiting", "unstarted", 1000);
  const read = await requestedIssue(tab.nerve, () => member.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  read.call.answer(json(200, issuePageOf([listIssue("x", 1, { project_id: "A", state_id: state.id })])));
  await until(() => tab.nerve.calls.some((call) => call.path === "/api/v0/projects/A/states"), "states directory");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  tab.nerve.calls.find((call) => call.path === "/api/v0/projects/A/states")!.answer(json(200, { data: [state] }));
  await read.promise;
  expect(member.value).toBeUndefined();
  expect(tab.app.issue.issues.held("x")).toBeUndefined();
});
