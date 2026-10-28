/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { answered, json } from "@/lib/auth/fake-nerve";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
  await answered(
    tab.nerve,
    () => tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([issueOf("R0")]),
    "list"
  );
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
it("waitsForRealResponseThenAddsCanonicalGroupAndCountOnce", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssues;
  const request = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "First" }), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  expect(request.call.body).toEqual({ name: "First" });
  expect(store.getIssueIds()).toEqual(["R0"]);
  expect(store.total).toBe(1);
  expect(app.issue.issues.getIssueById("R1")).toBeUndefined();
  request.call.answer(json(201, issueDetailOf("R1")));
  await request.promise;
  expect(store.getIssueIds()).toEqual(["R1", "R0"]);
  expect(store.total).toBe(2);
  expect(app.issue.issues.getIssueDetailById("R1")).toEqual(issueDetailOf("R1"));
});
it("refusedCreationLeavesCanonicalGroupAndCountUnchanged", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssues;
  const request = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "First" }), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  const rejected = expect(request.promise).rejects.toBeDefined();
  request.call.answer(json(403, { code: "forbidden" }));
  await rejected;
  expect(store.getIssueIds()).toEqual(["R0"]);
  expect(store.total).toBe(1);
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual(["R0"]);
});
it("lateACreationUpdatesItsBoundListWithoutEnteringSelectedB", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssues;
  const listA = store.currentList!;
  const request = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "First" }), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  runInAction(() => app.router.setQuery({ workspaceSlug: "ws", projectId: "B" }));
  await answered(
    nerve,
    () => store.fetchIssues("ws", "B", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/B/issues"],
    issuePageOf([issueOf("B0", "B")]),
    "B list"
  );
  request.call.answer(json(201, issueDetailOf("A1")));
  await request.promise;
  expect(store.getIssueIds()).toEqual(["B0"]);
  expect(store.total).toBe(1);
  expect(listA.reconciled.value?.total).toBe(2);
  expect(app.issue.issues.getIssueById("A1")?.project_id).toBe("A");
});
