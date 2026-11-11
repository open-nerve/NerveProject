/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The priority work items of cycles, for the card of the cycle in progress (P5b spec 2.7, R7; re-66 M4): a list store
// of their own beside the cycle page's, made once by the root; each cycle's one binding, registered once in the root;
// the explicit whole query, none of a page's preferences in it; the next page by its cursor after the rows held, in
// nerve's order; a confirmed write decides by the cycle and the priorities whether a row stays; and nothing of a read
// answered once the root has retired.
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CyclePreferences, Issue } from "@nerve/api-client";
import { answered, json } from "@/lib/auth/fake-nerve";
import { preferencesOf } from "@/store/project/fake-projects";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";

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
/** The query of a cycle's priority work items, as nerve is asked it. */
const priorityQueryOf = (cycleID: string) => ({
  cycle_id: cycleID,
  priority: "urgent,high",
  order_by: "-created_at",
  sub_issue: "true",
  limit: "30",
});
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

/** A work item of the cycle, of a priority, made at a time: the tab's order is the newest first. */
const inCycle = (id: string, priority: Issue["priority"], created_at: string, cycle_id = C1): Issue => ({
  ...issueOf(id),
  priority,
  cycle_id,
  created_at,
});
const a = inCycle("a", "high", "2026-11-01T00:00:00Z");
const b = inCycle("b", "urgent", "2026-11-03T00:00:00Z");
const c = inCycle("c", "urgent", "2026-10-30T00:00:00Z");
/** The cycle's priority work items: what the tab shows of them, as the store gives it. */
const shown = (cycleID = C1) => tab.app.issue.cyclePriorityIssues.view("ws", pageOf(cycleID));
/** The first page of the cycle's priority work items, as nerve answers it. */
async function firstPage(rows: Issue[], next: string | null = null, cycleID = C1) {
  const read = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.cyclePriorityIssues.fetchIssues("ws", pageOf(cycleID)),
    ["GET", "/api/v0/projects/A/issues"]
  );
  read.call.answer(json(200, { ...issuePageOf(rows, next), total: rows.length + (next ? 1 : 0) }));
  await read.promise;
  return read.call;
}

it("reads a cycle's urgent and high work items by their own whole query, none of a page's preferences in it", async () => {
  const { app, nerve } = tab;
  // the cycle page's and the project page's preferences both say other terms: a query that borrowed them shows it
  const { filters: _none, ...defaults } = preferencesOf();
  const cyclePreferences: CyclePreferences = {
    filters: { label_id: ["l-1"] },
    display_filters: { ...defaults.display_filters, group_by: "state_id", order_by: "sort_order", sub_issue: false },
    display_properties: defaults.display_properties,
  };
  await answered(
    nerve,
    () => app.issue.cycleIssuesFilter.fetchFilters("ws", C1),
    ["GET", `/api/v0/me/cycles/${C1}/preferences`],
    cyclePreferences,
    "the cycle page's preferences"
  );
  runInAction(() => {
    app.issue.projectIssuesFilter.filters.A = {
      richFilters: { and: [{ priority__in: "low" }] },
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: { layout: "list", group_by: "state", order_by: "-updated_at" },
    };
  });
  expect(shown()).toBeUndefined();
  const call = await firstPage([b, a]);
  expect(call.query).toEqual(priorityQueryOf(C1));
  expect(shown()).toEqual({ total: 2, issueIds: ["b", "a"], more: false, error: undefined });
});

it("leaves the cycle page's list, and the project page's, as they are (R7)", async () => {
  const { app, nerve } = tab;
  const page = app.issue.cycleIssues;
  await answered(
    nerve,
    () => page.fetchIssues("ws", pageOf(C1), "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([inCycle("low", "low", "2026-11-02T00:00:00Z"), a]),
    "the cycle page's list"
  );
  const current = page.currentList;
  const value = current?.value;
  await firstPage([b, a]);
  expect(page.currentList).toBe(current);
  expect(page.currentList?.value).toBe(value);
  expect(page.getIssueIds()?.toSorted()).toEqual(["a", "low"]);
  expect(app.issue.projectIssues.currentList).toBeUndefined();
  expect(shown()?.issueIds).toEqual(["b", "a"]);
});

it("keeps one binding per cycle, registered once in the root, however often it is read", async () => {
  const registered = vi.spyOn(tab.app.issue, "registerList");
  await firstPage([b, a]);
  await firstPage([b]);
  expect(registered).toHaveBeenCalledTimes(1);
  expect(shown()?.issueIds).toEqual(["b"]);
  const other = await firstPage([inCycle("d", "high", "2026-11-04T00:00:00Z", C2)], null, C2);
  expect(other.query).toEqual(priorityQueryOf(C2));
  expect(registered).toHaveBeenCalledTimes(2);
  expect(shown(C2)?.issueIds).toEqual(["d"]);
  expect(shown()?.issueIds).toEqual(["b"]);
});

it("reads the next page by its cursor, after the rows it has, in nerve's order", async () => {
  const { app, nerve } = tab;
  const store = app.issue.cyclePriorityIssues;
  // no cycle read yet: no next page to read
  await expect(store.fetchNextIssues("ws", pageOf(C1))).resolves.toBeUndefined();
  expect(nerve.calls).toEqual([]);
  await firstPage([b, a], "next-1");
  expect(shown()).toEqual({ total: 3, issueIds: ["b", "a"], more: true, error: undefined });
  const next = await requestedIssue(nerve, () => store.fetchNextIssues("ws", pageOf(C1)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(next.call.query).toEqual({ ...priorityQueryOf(C1), cursor: "next-1" });
  next.call.answer(json(200, { ...issuePageOf([c]), total: 3 }));
  await next.promise;
  expect(shown()).toEqual({ total: 3, issueIds: ["b", "a", "c"], more: false, error: undefined });
  // the last page read: none left
  nerve.calls.length = 0;
  await expect(store.fetchNextIssues("ws", pageOf(C1))).resolves.toBeUndefined();
  expect(nerve.calls).toEqual([]);
});

it("keeps a row while nerve confirms it urgent or high in the cycle, and lets it go once it is neither", async () => {
  const { app, nerve } = tab;
  await firstPage([b, a]);
  const change = await requestedIssue(
    nerve,
    () => app.issue.updateIssue("a", () => ({ priority: "low" }), undefined, app.issue.scope("ws", "A")),
    ["PATCH", "/api/v0/issues/a"]
  );
  change.call.answer(json(200, { ...a, priority: "low", updated_at: "2026-11-05T00:00:00Z" }));
  await change.promise;
  expect(shown()?.issueIds).toEqual(["b"]);
  expect(shown()?.total).toBe(1);
});

it("writes nothing of an answer that comes once the root has retired", async () => {
  const { app, nerve } = tab;
  const read = await requestedIssue(nerve, () => app.issue.cyclePriorityIssues.fetchIssues("ws", pageOf(C1)), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  app.issue.retire();
  read.call.answer(json(200, issuePageOf([b, a])));
  await read.promise.catch(() => undefined);
  expect(shown()).toBeUndefined();
  expect(app.issue.issues.held("b")).toBeUndefined();
  await expect(app.issue.cyclePriorityIssues.fetchIssues("ws", pageOf(C1))).rejects.toThrow();
  expect(nerve.calls).toHaveLength(1);
});
