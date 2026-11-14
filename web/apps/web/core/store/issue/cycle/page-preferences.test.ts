/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The cycle page's own preferences (P5b spec 2.4), by cycle id, through the issue root's store and a fake nerve.
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CyclePreferences } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { json, problem } from "@/lib/auth/fake-nerve";
import { preferencesOf } from "@/store/project/fake-projects";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";
import { issueStoreTab, requestedIssue } from "../fake-issues";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

/** A cycle page's preferences as nerve answers them; its defaults are the project page's list defaults. */
function cyclePreferencesOf(fields: Partial<CyclePreferences> = {}): CyclePreferences {
  const { filters, display_filters, display_properties } = preferencesOf();
  return { filters, display_filters, display_properties, ...fields };
}
const read = (cycleId: string): Endpoint => ["GET", `/api/v0/me/cycles/${cycleId}/preferences`];
const write = (cycleId: string): Endpoint => ["PATCH", `/api/v0/me/cycles/${cycleId}/preferences`];
const high = { and: [{ priority__in: "high" }] };
const low = { and: [{ priority__in: "low" }] };

async function loaded(cycleId: string, value: CyclePreferences) {
  const store = tab.app.issue.cycleIssuesFilter;
  const loading = await requestedIssue(tab.nerve, () => store.fetchFilters("ws", cycleId), read(cycleId));
  loading.call.answer(json(200, value));
  await loading.promise;
  return store;
}

it("reads a cycle page's preferences by cycle id and shows nerve's words as the project page does", async () => {
  const { app, nerve } = tab;
  const store = await loaded("C1", cyclePreferencesOf());
  expect(store.getIssueFilters("C1")).toMatchObject({
    richFilters: {},
    displayFilters: {
      layout: "list",
      group_by: null,
      sub_group_by: null,
      order_by: "-created_at",
      sub_issue: true,
      show_empty_groups: true,
      calendar: { layout: "month", show_weekends: false },
    },
    displayProperties: cyclePreferencesOf().display_properties,
    kanbanFilters: { group_by: [], sub_group_by: [] },
  });
  const saved = cyclePreferencesOf({
    filters: { priority: ["high"] },
    display_filters: {
      ...cyclePreferencesOf().display_filters,
      layout: "kanban",
      group_by: "state_id",
      sub_group_by: "priority",
      order_by: "-state_name",
      calendar: { layout: "week", show_weekends: true },
    },
  });
  await loaded("C1", saved);
  expect(store.getIssueFilters("C1")).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "kanban", group_by: "state", sub_group_by: "priority", order_by: "-state__name" },
  });
  const project = await requestedIssue(nerve, () => app.issue.projectIssuesFilter.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  project.call.answer(json(200, preferencesOf(saved)));
  await project.promise;
  expect(store.getIssueFilters("C1")).toEqual(app.issue.projectIssuesFilter.getIssueFilters("A"));
});

it("sends only the changed field and shows it once nerve confirms it", async () => {
  const { nerve } = tab;
  const initial = cyclePreferencesOf();
  const store = await loaded("C1", initial);
  const filters = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", high), write("C1"));
  expect(filters.call.body).toEqual({ filters: { priority: ["high"] } });
  expect(store.getIssueFilters("C1")?.richFilters).toEqual({});
  filters.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["high"] } })));
  await filters.promise;
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(high);
  const display = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "C1", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false }),
    write("C1")
  );
  expect(display.call.body).toEqual({ display_properties: { ...initial.display_properties, priority: false } });
  expect(store.getIssueFilters("C1")?.displayProperties?.priority).toBe(true);
  const confirmed = { ...initial.display_properties, priority: false, labels: false };
  display.call.answer(json(200, cyclePreferencesOf({ display_properties: confirmed })));
  await display.promise;
  expect(store.getIssueFilters("C1")?.displayProperties).toEqual(confirmed);
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(high);
});

it("changes nothing on a refusal and hands the refusal to the caller", async () => {
  const { nerve } = tab;
  const store = await loaded("C1", cyclePreferencesOf({ filters: { priority: ["high"] } }));
  const filters = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", low), write("C1"));
  filters.call.answer(problem(403, "forbidden"));
  await expect(filters.promise).rejects.toMatchObject({ status: 403, problem: { code: "forbidden" } });
  await expect(filters.promise).rejects.toBeInstanceOf(ApiError);
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(high);
  const display = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "C1", EIssueFilterType.DISPLAY_FILTERS, { layout: "kanban" }),
    write("C1")
  );
  display.call.answer(problem(404, "planning.cycle_not_found"));
  await expect(display.promise).rejects.toMatchObject({ problem: { code: "planning.cycle_not_found" } });
  expect(store.getIssueFilters("C1")?.displayFilters?.layout).toBe("list");
});

it("keeps a confirmed change when an older read answers after it", async () => {
  const { nerve } = tab;
  const initial = cyclePreferencesOf();
  const store = await loaded("C1", initial);
  const older = await requestedIssue(nerve, () => store.fetchFilters("ws", "C1"), read("C1"));
  const patch = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "C1", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false }),
    write("C1")
  );
  patch.call.answer(
    json(200, cyclePreferencesOf({ display_properties: { ...initial.display_properties, priority: false } }))
  );
  await patch.promise;
  older.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["low"] } })));
  await older.promise;
  expect(store.getIssueFilters("C1")?.displayProperties?.priority).toBe(false);
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(low);
});

it("keeps two cycles' preferences apart: their own values, queues and reads", async () => {
  const { nerve } = tab;
  const store = await loaded("C1", cyclePreferencesOf());
  await loaded("C2", cyclePreferencesOf({ filters: { priority: ["low"] } }));
  expect(store.getIssueFilters("C1")?.richFilters).toEqual({});
  expect(store.getIssueFilters("C2")?.richFilters).toEqual(low);
  const first = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", high), write("C1"));
  const second = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C2", {}), write("C2"));
  second.call.answer(json(200, cyclePreferencesOf()));
  await second.promise;
  expect(store.getIssueFilters("C2")?.richFilters).toEqual({});
  expect(store.getIssueFilters("C1")?.richFilters).toEqual({});
  const c2Read = await requestedIssue(nerve, () => store.fetchFilters("ws", "C2"), read("C2"));
  first.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["high"] } })));
  await first.promise;
  c2Read.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["urgent"] } })));
  await c2Read.promise;
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(high);
  expect(store.getIssueFilters("C2")?.richFilters).toEqual({ and: [{ priority__in: "urgent" }] });
});

it("writes no late answer once its generation retires, and sends nothing after", async () => {
  const { app, nerve } = tab;
  const store = await loaded("C1", cyclePreferencesOf());
  const late = await requestedIssue(nerve, () => store.fetchFilters("ws", "C2"), read("C2"));
  const change = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", high), write("C1"));
  app.dispose();
  late.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["low"] } })));
  change.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["high"] } })));
  await expect(late.promise).rejects.toBeInstanceOf(SessionChangedError);
  await expect(change.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store.getIssueFilters("C2")).toBeUndefined();
  expect(store.getIssueFilters("C1")?.richFilters).toEqual({});
  await expect(store.updateFilterExpression("ws", "C1", high)).rejects.toBeInstanceOf(SessionChangedError);
  expect(nerve.calls).toHaveLength(3);
});

it("replaces an unread cycle's filters with one PATCH and seeds the page from nerve's answer", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.cycleIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", high), write("C1"));
  expect(patch.call.body).toEqual({ filters: { priority: ["high"] } });
  const answer = cyclePreferencesOf({
    filters: { priority: ["high"] },
    display_filters: { ...cyclePreferencesOf().display_filters, layout: "spreadsheet" },
  });
  patch.call.answer(json(200, answer));
  await patch.promise;
  expect(nerve.calls).toHaveLength(1);
  expect(store.getIssueFilters("C1")).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "spreadsheet" },
    displayProperties: answer.display_properties,
  });
});

it("keeps an unread cycle's replaced filters when a read sent after the change answers older", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.cycleIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", high), write("C1"));
  const later = await requestedIssue(nerve, () => store.fetchFilters("ws", "C1"), read("C1"));
  patch.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["high"] } })));
  await patch.promise;
  const properties = { ...cyclePreferencesOf().display_properties, state: false };
  later.call.answer(json(200, cyclePreferencesOf({ display_properties: properties })));
  await later.promise;
  expect(nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(high);
  expect(store.getIssueFilters("C1")?.displayProperties).toEqual(properties);
});

it("follows no route: each call names its cycle and what it holds stays whatever the route", async () => {
  const { app, nerve } = tab;
  const store = app.issue.cycleIssuesFilter;
  runInAction(() => app.router.setQuery({ workspaceSlug: "ws", projectId: "A", cycleId: "C2" }));
  await loaded("C1", cyclePreferencesOf({ filters: { priority: ["high"] } }));
  expect(store.getIssueFilters("C2")).toBeUndefined();
  const shown = store.getIssueFilters("C1");
  expect(shown?.richFilters).toEqual(high);
  runInAction(() => app.router.setQuery({ workspaceSlug: "ws", projectId: "B", cycleId: "C1" }));
  expect(store.getIssueFilters("C1")).toEqual(shown);
  expect(store.getIssueFilters("C2")).toBeUndefined();
  runInAction(() => app.router.setQuery({ workspaceSlug: "other", cycleId: "C3" }));
  expect(store.getIssueFilters("C1")).toEqual(shown);
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "C1", low), write("C1"));
  patch.call.answer(json(200, cyclePreferencesOf({ filters: { priority: ["low"] } })));
  await patch.promise;
  expect(store.getIssueFilters("C1")?.richFilters).toEqual(low);
  expect(store.getIssueFilters("C3")).toBeUndefined();
  expect("issueFilters" in store).toBe(false);
});

it("keeps group collapse locally under the cycle page's store type", async () => {
  const { app, nerve } = tab;
  const alice = accountOf("alice");
  const entry = (key: string, group: string) => ({
    key,
    workspaceSlug: "ws",
    viewId: "C1",
    userId: alice.id,
    filters: { kanban_filters: { group_by: [group], sub_group_by: [] } },
  });
  const data = new Map([["issue_local_filters", JSON.stringify([entry("PROJECT", "other"), entry("CYCLE", "mine")])]]);
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => data.set(key, value),
    },
  });
  await seedAccount(nerve, app.user, alice);
  const store = await loaded("C1", cyclePreferencesOf());
  expect(store.getIssueFilters("C1")?.kanbanFilters?.group_by).toEqual(["mine"]);
  await store.updateFilters("ws", "C1", EIssueFilterType.KANBAN_FILTERS, { group_by: ["changed"], sub_group_by: [] });
  expect(JSON.parse(data.get("issue_local_filters") ?? "[]")).toEqual([
    entry("PROJECT", "other"),
    { ...entry("CYCLE", "mine"), filters: { kanban_filters: { group_by: ["changed"], sub_group_by: [] } } },
  ]);
});
