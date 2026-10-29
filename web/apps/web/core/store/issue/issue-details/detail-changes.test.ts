/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { issueDetailOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { IssueListBinding } from "../helpers/issue-list-binding";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const parent = { ...issueDetailOf("parent"), sub_issues_count: 8 };
const child = { ...issueDetailOf("child"), parent_id: "parent" };
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  for (const issue of [parent, child, issueDetailOf("other")])
    tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path === "/api/v0/projects/A/states") call.answer(json(200, { data: [] }));
  };
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
it("a core parent PATCH refreshes both old and new parents without guessing either count", async () => {
  const root = tab.app.issue;
  const sent = await requestedIssue(tab.nerve, () => root.updateIssue("child", () => ({ parent_id: "other" })), [
    "PATCH",
    "/api/v0/issues/child",
  ]);
  expect(root.issues.getIssueById("parent")?.sub_issues_count).toBe(8);
  sent.call.answer(json(200, { ...child, parent_id: "other", updated_at: "2026-10-29T00:00:00Z" }));
  await until(() => tab.nerve.to("/api/v0/issues/parent").length > 0 || sent.result.settled, "old parent read");
  expect(tab.nerve.to("/api/v0/issues/parent")).toHaveLength(1);
  expect(tab.nerve.to("/api/v0/issues/other")).toHaveLength(1);
  expect(root.issues.getIssueById("parent")).toBeUndefined();
  tab.nerve
    .to("/api/v0/issues/parent")[0]!
    .answer(json(200, { ...parent, sub_issues_count: 3, updated_at: "2026-10-29T00:00:00Z" }));
  tab.nerve
    .to("/api/v0/issues/other")[0]!
    .answer(json(200, { ...issueDetailOf("other"), sub_issues_count: 7, updated_at: "2026-10-29T00:00:00Z" }));
  await sent.promise;
  expect(root.issues.getIssueById("parent")?.sub_issues_count).toBe(3);
  expect(root.issues.getIssueById("other")?.sub_issues_count).toBe(7);
});
it("a batch and the child's later nullable parent PATCH share the same queue", async () => {
  const root = tab.app.issue;
  const batch = await requestedIssue(
    tab.nerve,
    () => root.issueDetail.subIssues.createSubIssues("ws", "A", "parent", ["child"]),
    ["POST", "/api/v0/issues/parent/sub-issues"]
  );
  const detach = track(root.issueDetail.subIssues.removeSubIssue("ws", "A", "parent", "child"));
  await vi.advanceTimersByTimeAsync(20);
  expect(tab.nerve.to("/api/v0/issues/child")).toHaveLength(0);
  batch.call.answer(json(200, { data: [child] }));
  await until(() => tab.nerve.to("/api/v0/issues/parent").length > 0, "parent read");
  tab.nerve.to("/api/v0/issues/parent")[0]!.answer(problem(500, "unavailable"));
  await batch.promise;
  expect(root.issueDetail.subIssues.subIssuesByIssueId("parent")).toEqual(["child"]);
  await until(() => tab.nerve.to("/api/v0/issues/child").length > 0, "child PATCH");
  const patch = tab.nerve.to("/api/v0/issues/child")[0]!;
  expect(patch.body).toEqual({ parent_id: null });
  patch.answer(json(200, { ...child, parent_id: null, updated_at: "2026-10-29T00:00:00Z" }));
  await until(
    () => tab.nerve.to("/api/v0/issues/parent").length === 2 || detach.settled,
    "detached parent verification"
  );
  expect(tab.nerve.to("/api/v0/issues/parent")).toHaveLength(2);
  tab.nerve
    .to("/api/v0/issues/parent")[1]!
    .answer(json(200, { ...parent, sub_issues_count: 5, updated_at: "2026-10-29T00:00:00Z" }));
  await until(() => detach.settled, "detached");
  expect(detach.error).toBeUndefined();
  expect(root.issueDetail.subIssues.subIssuesByIssueId("parent")).toEqual([]);
  expect(tab.nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
});
it("a child deletion uses the root's subtree tombstones and verifies its parent's count", async () => {
  const root = tab.app.issue;
  root.issues.acceptRead({ ...issueDetailOf("grandchild"), parent_id: "child" }, root.captureRead());
  const sent = await requestedIssue(
    tab.nerve,
    () => root.issueDetail.subIssues.deleteSubIssue("ws", "A", "parent", "child"),
    ["DELETE", "/api/v0/issues/child"]
  );
  sent.call.answer(noContent());
  await until(() => tab.nerve.to("/api/v0/issues/parent").length > 0 || sent.result.settled, "deleted child's parent");
  expect(tab.nerve.to("/api/v0/issues/parent")).toHaveLength(1);
  tab.nerve
    .to("/api/v0/issues/parent")[0]!
    .answer(json(200, { ...parent, sub_issues_count: 4, updated_at: "2026-10-29T00:00:00Z" }));
  await sent.promise;
  expect(root.issues.held("child")).toBeNull();
  expect(root.issues.held("grandchild")).toBeNull();
  expect(root.issues.getIssueById("parent")?.sub_issues_count).toBe(4);
});
it("a loaded child collection gets a fresh read after deletion invalidates its proof", async () => {
  const root = tab.app.issue;
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (call.path === "/api/v0/projects/A/states") call.answer(json(200, { data: [] }));
    if (call.path === "/api/v0/issues/parent") call.answer(json(200, parent));
  };
  const reading = await requestedIssue(
    tab.nerve,
    () => root.issueDetail.subIssues.fetchSubIssues("ws", "A", "parent"),
    ["GET", "/api/v0/issues/parent/sub-issues"]
  );
  reading.call.answer(json(200, { data: [child] }));
  await reading.promise;
  const deleting = await requestedIssue(tab.nerve, () => root.deleteIssue("child", root.scope("ws", "A")), [
    "DELETE",
    "/api/v0/issues/child",
  ]);
  deleting.call.answer(noContent());
  await until(
    () => tab.nerve.to("/api/v0/issues/parent/sub-issues").length === 2 || deleting.result.settled,
    "fresh children"
  );
  expect(tab.nerve.to("/api/v0/issues/parent/sub-issues")).toHaveLength(2);
  tab.nerve.to("/api/v0/issues/parent/sub-issues")[1]!.answer(json(200, { data: [] }));
  await deleting.promise;
  expect(root.issueDetail.subIssues.subIssuesByIssueId("parent")).toEqual([]);
});

it("a confirmed batch with an unknown old parent marks every covered group stale without guessing counts", async () => {
  const root = tab.app.issue;
  const list = new IssueListBinding(root, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: tab.workspace.id,
    projectID: "A",
    query: { order_by: "sort_order", group_by: "priority" },
  });
  const reading = await requestedIssue(tab.nerve, () => list.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  reading.call.answer(
    json(200, {
      total: 12,
      groups: [
        { key: "none", total: 5, data: [parent], next_cursor: "more-none" },
        { key: "high", total: 7, data: [], next_cursor: "more-high" },
      ],
    })
  );
  await reading.promise;
  const batch = await requestedIssue(
    tab.nerve,
    () => root.issueDetail.subIssues.createSubIssues("ws", "A", "parent", ["unread-child"]),
    ["POST", "/api/v0/issues/parent/sub-issues"]
  );
  expect(list.group("high")?.stale).toBe(false);
  batch.call.answer(json(200, { data: [{ ...child, id: "unread-child" }] }));
  await until(() => tab.nerve.to("/api/v0/issues/parent").length > 0, "parent confirmation read");
  expect(list.group("high")?.stale).toBe(true);
  expect(list.group("high")?.total).toBe(7);
  tab.nerve.to("/api/v0/issues/parent")[0]!.answer(problem(500, "unavailable"));
  await batch.promise;
});
