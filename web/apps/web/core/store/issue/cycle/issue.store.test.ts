/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The cycle page's list (P5b spec 2.1, 2.5, R1, R4): its project's list under the page's own preferences and the cycle.
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CyclePreferences, Issue } from "@nerve/api-client";
import { answered, json } from "@/lib/auth/fake-nerve";
import { loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const pageOf = (cycleID: string) => ({ projectID: "A", cycleID });
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
  // The project page's own preferences differ, so a query that borrowed them would show it.
  runInAction(() => {
    tab.app.issue.projectIssuesFilter.filters.A = {
      richFilters: { and: [{ priority__in: "low" }] },
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: { layout: "list", group_by: "state", order_by: "-updated_at" },
    };
  });
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

/** A cycle page's preferences as nerve answers them; its defaults are the project page's list defaults. */
function cyclePreferencesOf(fields: Partial<CyclePreferences> = {}): CyclePreferences {
  const { filters, display_filters, display_properties } = preferencesOf();
  return { filters, display_filters, display_properties, ...fields };
}
const highFirst = (cycleID: string) =>
  answered(
    tab.nerve,
    () => tab.app.issue.cycleIssuesFilter.fetchFilters("ws", cycleID),
    ["GET", `/api/v0/me/cycles/${cycleID}/preferences`],
    cyclePreferencesOf({
      filters: { priority: ["high"] },
      display_filters: { ...cyclePreferencesOf().display_filters, order_by: "sort_order" },
    }),
    "cycle preferences"
  );
const inCycle = (id: string, cycle_id: string | null = C1): Issue => ({ ...issueOf(id), cycle_id });
/** The ids the cycle page lists, in id order: the tests here are about which work items, not their order. */
const ids = () => tab.app.issue.cycleIssues.getIssueIds()?.toSorted();

it("lists the cycle as its project's list under the cycle page's preferences, never the project page's", async () => {
  const { app, nerve } = tab;
  await highFirst(C1);
  const store = app.issue.cycleIssues;
  const first = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(first.call.query).toEqual({
    priority: "high",
    order_by: "sort_order",
    sub_issue: "true",
    limit: "20",
    cycle_id: C1,
  });
  expect(store.currentList?.scope).toMatchObject({
    scope: "project",
    workspaceSlug: "ws",
    projectID: "A",
    query: { priority: ["high"], cycle_id: [C1] },
  });
  first.call.answer(json(200, issuePageOf([inCycle("a"), inCycle("b")])));
  await first.promise;
  expect(ids()).toEqual(["a", "b"]);
  expect(store.total).toBe(2);
  expect(app.issue.projectIssues.currentList).toBeUndefined();
  // the page offers new and changed work items as its store; an ended or archived cycle's roots narrow it (re-66 M2)
  expect(store.viewFlags).toEqual({ enableQuickAdd: true, enableIssueCreation: true, enableInlineEditing: true });
});

it("keeps one binding per cycle, so a late answer for the cycle left does not enter the cycle shown", async () => {
  const { app, nerve } = tab;
  const store = app.issue.cycleIssues;
  const one = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  const listOne = store.currentList!;
  const two = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(C2), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(two.call.query.cycle_id).toBe(C2);
  expect(store.currentList).not.toBe(listOne);
  two.call.answer(json(200, issuePageOf([inCycle("b", C2)])));
  await two.promise;
  one.call.answer(json(200, issuePageOf([inCycle("a")])));
  await one.promise;
  expect(ids()).toEqual(["b"]);
  expect(store.isPreferenceQueryCurrent(pageOf(C2))).toBe(true);
  expect(store.isPreferenceQueryCurrent(pageOf(C1))).toBe(false);
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([inCycle("a")]),
    "back to the first cycle"
  );
  expect(store.currentList).toBe(listOne);
  expect(ids()).toEqual(["a"]);
});

it("counts a cycle page's work items as nerve does, only while the current list is that page's", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.cycleIssues;
  expect(store.pageTotal(pageOf(C1)), "nothing read").toBeUndefined();
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    { ...issuePageOf([inCycle("a")]), total: 7 },
    "the first cycle's list"
  );
  expect(store.pageTotal(pageOf(C1)), "nerve's count, though it sent one row").toBe(7);
  expect(store.pageTotal(pageOf(C2)), "another cycle's page").toBeUndefined();
  expect(store.pageTotal({ projectID: "B", cycleID: C1 }), "the cycle under another project").toBeUndefined();
  const two = await requestedIssue(nerve, () => store.fetchIssues("ws", pageOf(C2), "init-loader", { limit: 20 }), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(store.pageTotal(pageOf(C1)), "the first cycle's, its list no longer current").toBeUndefined();
  expect(store.pageTotal(pageOf(C2)), "the second cycle's, not answered yet").toBeUndefined();
  two.call.answer(json(200, { ...issuePageOf([inCycle("b", C2)]), total: 1 }));
  await two.promise;
  expect(store.pageTotal(pageOf(C2))).toBe(1);
});

it("reads the next page only for the cycle page whose list is current", async () => {
  const { app, nerve } = tab;
  const store = app.issue.cycleIssues;
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 1 }),
    ["GET", "/api/v0/projects/A/issues"],
    { ...issuePageOf([inCycle("a")], "next-a"), total: 2 },
    "first page"
  );
  nerve.calls.length = 0;
  await expect(store.fetchNextIssues("ws", pageOf(C2))).resolves.toBeUndefined();
  await expect(store.fetchNextIssues("ws", { projectID: "B", cycleID: C1 })).resolves.toBeUndefined();
  expect(nerve.calls).toEqual([]);
  const next = await requestedIssue(nerve, () => store.fetchNextIssues("ws", pageOf(C1)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(next.call.query).toMatchObject({ cycle_id: C1, cursor: "next-a" });
  next.call.answer(json(200, { ...issuePageOf([inCycle("b")]), total: 2 }));
  await next.promise;
  expect(ids()).toEqual(["a", "b"]);
});

it("knows when a confirmed change of the cycle page's preferences leaves its list behind, and reads it again", async () => {
  const { app, nerve } = tab;
  await highFirst(C1);
  const store = app.issue.cycleIssues;
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([inCycle("a")]),
    "list"
  );
  expect(store.isPreferenceQueryCurrent(pageOf(C1))).toBe(true);
  await answered(
    nerve,
    () => app.issue.cycleIssuesFilter.updateFilterExpression("ws", C1, { and: [{ priority__in: "urgent" }] }),
    ["PATCH", `/api/v0/me/cycles/${C1}/preferences`],
    cyclePreferencesOf({ filters: { priority: ["urgent"] } }),
    "filters"
  );
  expect(store.isPreferenceQueryCurrent(pageOf(C1))).toBe(false);
  const again = await requestedIssue(nerve, () => store.fetchIssuesWithExistingPagination("ws", pageOf(C1)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(again.call.query).toMatchObject({ priority: "urgent", limit: "20", cycle_id: C1 });
  again.call.answer(json(200, issuePageOf([])));
  await again.promise;
  expect(store.isPreferenceQueryCurrent(pageOf(C1))).toBe(true);
});

it("creates in the cycle's project with the cycle in the one request, and elsewhere without it", async () => {
  const { app, nerve } = tab;
  await loadProjects(nerve, app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  nerve.calls.length = 0;
  const store = app.issue.cycleIssues;
  await answered(
    nerve,
    () => store.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([inCycle("a")]),
    "list"
  );
  const here = await requestedIssue(nerve, () => store.createIssue("ws", "A", { name: "Here" }, pageOf(C1)), [
    "POST",
    "/api/v0/projects/A/issues",
  ]);
  expect(here.call.body).toEqual({ name: "Here", cycle_id: C1 });
  here.call.answer(json(201, { ...issueDetailOf("here"), sort_order: 10, cycle_id: C1 }));
  await here.promise;
  expect(ids()).toEqual(["a", "here"]);
  const elsewhere = await requestedIssue(nerve, () => store.createIssue("ws", "B", { name: "Elsewhere" }, pageOf(C1)), [
    "POST",
    "/api/v0/projects/B/issues",
  ]);
  expect(elsewhere.call.body).toEqual({ name: "Elsewhere" });
  elsewhere.call.answer(json(201, issueDetailOf("elsewhere", "B")));
  await elsewhere.promise;
  expect(ids()).toEqual(["a", "here"]);
  expect(nerve.calls).toHaveLength(3);
});
