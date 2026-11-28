/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's pages' list (M7/P6b spec 2.3, 2.5, R1, R4): the workspace's list under the page's preferences, flat,
// a default view's own term, `me` in it, last in its query and in none of its page's words; a page of a workspace the
// store does not give reads nothing. A write anywhere it covers comes into it: on "Assigned to me" a row whose assignee
// another write changed leaves it at once, the project page's write as well as its own (re-97 I1). The stores and HTTP
// are real.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueFilterType } from "@nerve/constants";
import { json } from "@/lib/auth/fake-nerve";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { workspaceViewOf } from "@/store/views/fake-views";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { workspacePageId, workspacePageWords } from "./page-id";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  tab.nerve.calls.length = 0;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const A = "019a0000-0000-7000-8000-000000000001";
const LIST: [string, string] = ["GET", "/api/v0/workspaces/ws/issues"];
const issues = () => tab.app.issue.globalIssues;
const filters = () => tab.app.issue.globalIssuesFilter;
/** A default view's page, its preferences read: its page id. */
async function page(key: string) {
  const id = workspacePageId("ws", key);
  await filters().fetchFilters("ws", id);
  return id;
}
async function listed(globalViewId: string, rows = [issueOf("x")]) {
  const read = await requestedIssue(tab.nerve, () => issues().fetchIssues("ws", globalViewId, { limit: 50 }), LIST);
  read.call.answer(json(200, issuePageOf(rows)));
  await read.promise;
  return read.call;
}

it("reads 'Assigned to me' with the page's filters and its own term last, flat, its page offering no assignee", async () => {
  const id = await page("assigned");
  await filters().updateFilterExpression("ws", id, { and: [{ priority__in: "high" }] });
  const call = await listed("assigned");
  expect(call.queryEntries).toEqual([
    ["priority", "high"],
    ["order_by", "-created_at"],
    ["sub_issue", "true"],
    ["assignee_id", "me"],
    ["limit", "50"],
  ]);
  expect(workspacePageWords("assigned")).not.toContain("assignee_id");
  expect(issues().viewFlags).toEqual({ enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: true });
});

it("reads a saved view's page with its draft's words and no term of its own", async () => {
  const view = workspaceViewOf(tab.workspace, "Mine", "actor", { filters: { project_id: [A] } });
  const read = await requestedIssue(tab.nerve, () => filters().fetchFilters("ws", view.id), [
    "GET",
    `/api/v0/views/${view.id}`,
  ]);
  read.call.answer(json(200, view));
  await read.promise;
  const call = await listed(view.id, []);
  expect(call.query).toEqual({ project_id: A, order_by: "-created_at", sub_issue: "true", limit: "50" });
});

it("reads nothing for a workspace the store does not give", async () => {
  await expect(issues().fetchIssues("gone", "all-issues", { limit: 50 })).resolves.toBeUndefined();
  expect(tab.nerve.calls).toEqual([]);
});

it("says whether its list was read with the page's preferences as they are now, its next page by cursor", async () => {
  const id = await page("created");
  expect(issues().isPreferenceQueryCurrent("ws", "created")).toBe(false);
  const read = await requestedIssue(tab.nerve, () => issues().fetchIssues("ws", "created", { limit: 1 }), LIST);
  read.call.answer(json(200, issuePageOf([issueOf("x")], "c1")));
  await read.promise;
  expect(issues().isPreferenceQueryCurrent("ws", "created")).toBe(true);
  expect(issues().isPreferenceQueryCurrent("beta", "created")).toBe(false);
  await expect(issues().fetchNextIssues("beta")).resolves.toBeUndefined();
  const next = await requestedIssue(tab.nerve, () => issues().fetchNextIssues("ws"), LIST);
  expect(next.call.query).toMatchObject({ created_by_id: "me", cursor: "c1" });
  next.call.answer(json(200, issuePageOf([])));
  await next.promise;
  await filters().updateFilters("ws", id, EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" });
  expect(issues().isPreferenceQueryCurrent("ws", "created")).toBe(false);
});

it.each([
  ["its own write", () => issues().updateIssue("ws", "A", "x", { assignee_ids: ["other"] })],
  ["the project page's", () => tab.app.issue.projectIssues.updateIssue("ws", "A", "x", { assignee_ids: ["other"] })],
])("drops a row of 'Assigned to me' once %s gives it to another, the total one less", async (_who, write) => {
  await page("assigned");
  await listed("assigned", [
    { ...issueOf("x"), assignee_ids: ["actor"] },
    { ...issueOf("y"), assignee_ids: ["actor"] },
  ]);
  expect([issues().getIssueIds(), issues().total]).toEqual([["x", "y"], 2]);
  const sent = await requestedIssue(tab.nerve, write, ["PATCH", "/api/v0/issues/x"]);
  sent.call.answer(json(200, { ...issueDetailOf("x"), assignee_ids: ["other"], updated_at: "2026-10-29T00:00:00Z" }));
  await sent.promise;
  expect([issues().getIssueIds(), issues().total]).toEqual([["y"], 1]);
});
