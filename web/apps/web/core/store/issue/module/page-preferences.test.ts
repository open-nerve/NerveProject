/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The module page's own preferences (P6b spec 2.4), by module id, through the issue root's store and a fake nerve.
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ModulePreferences } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import type { TWorkItemFilterExpression } from "@nerve/types";
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

/** A module page's preferences as nerve answers them; its defaults are the project page's list defaults. */
function modulePreferencesOf(fields: Partial<ModulePreferences> = {}): ModulePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters: {}, display_filters, display_properties, ...fields };
}
const read = (moduleId: string): Endpoint => ["GET", `/api/v0/me/modules/${moduleId}/preferences`];
const write = (moduleId: string): Endpoint => ["PATCH", `/api/v0/me/modules/${moduleId}/preferences`];
const high = { and: [{ priority__in: "high" }] };
const low = { and: [{ priority__in: "low" }] };

async function loaded(moduleId: string, value: ModulePreferences) {
  const store = tab.app.issue.moduleIssuesFilter;
  const loading = await requestedIssue(tab.nerve, () => store.fetchFilters("ws", moduleId), read(moduleId));
  loading.call.answer(json(200, value));
  await loading.promise;
  return store;
}

it("reads a module page's preferences by module id and shows nerve's words as the project page does", async () => {
  const { app, nerve } = tab;
  const store = await loaded("M1", modulePreferencesOf());
  expect(store.getIssueFilters("M1")).toMatchObject({
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
    displayProperties: modulePreferencesOf().display_properties,
    kanbanFilters: { group_by: [], sub_group_by: [] },
  });
  const saved = modulePreferencesOf({
    filters: { priority: ["high"] },
    display_filters: {
      ...modulePreferencesOf().display_filters,
      layout: "kanban",
      group_by: "state_id",
      sub_group_by: "priority",
      order_by: "-state_name",
      calendar: { layout: "week", show_weekends: true },
    },
  });
  await loaded("M1", saved);
  expect(store.getIssueFilters("M1")).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "kanban", group_by: "state", sub_group_by: "priority", order_by: "-state__name" },
  });
  const project = await requestedIssue(nerve, () => app.issue.projectIssuesFilter.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  project.call.answer(json(200, preferencesOf(saved)));
  await project.promise;
  expect(store.getIssueFilters("M1")).toEqual(app.issue.projectIssuesFilter.getIssueFilters("A"));
});

it("shows no cycle condition saved through the interface, and sends the web's conditions alone, never a module", async () => {
  const { nerve } = tab;
  const store = await loaded("M1", modulePreferencesOf({ filters: { cycle_id: ["none"], priority: ["high"] } }));
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
  // The web's words have no module (P6b R16); a condition on one, from wherever it came, is not sent.
  const withModule = { and: [{ priority__in: "low" }, { module_id__in: "M2" }] } as TWorkItemFilterExpression;
  const filters = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", withModule), write("M1"));
  expect(filters.call.body).toEqual({ filters: { priority: ["low"] } });
  filters.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["low"] } })));
  await filters.promise;
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(low);
});

it("sends only the changed field and shows it once nerve confirms it", async () => {
  const { nerve } = tab;
  const initial = modulePreferencesOf();
  const store = await loaded("M1", initial);
  const filters = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", high), write("M1"));
  expect(filters.call.body).toEqual({ filters: { priority: ["high"] } });
  expect(store.getIssueFilters("M1")?.richFilters).toEqual({});
  filters.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["high"] } })));
  await filters.promise;
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
  const display = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "M1", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false }),
    write("M1")
  );
  expect(display.call.body).toEqual({ display_properties: { ...initial.display_properties, priority: false } });
  expect(store.getIssueFilters("M1")?.displayProperties?.priority).toBe(true);
  const confirmed = { ...initial.display_properties, priority: false, labels: false };
  display.call.answer(json(200, modulePreferencesOf({ display_properties: confirmed })));
  await display.promise;
  expect(store.getIssueFilters("M1")?.displayProperties).toEqual(confirmed);
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
});

it("changes nothing on a refusal and hands the refusal to the caller", async () => {
  const { nerve } = tab;
  const store = await loaded("M1", modulePreferencesOf({ filters: { priority: ["high"] } }));
  const filters = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", low), write("M1"));
  filters.call.answer(problem(403, "forbidden"));
  await expect(filters.promise).rejects.toMatchObject({ status: 403, problem: { code: "forbidden" } });
  await expect(filters.promise).rejects.toBeInstanceOf(ApiError);
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
  const display = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "M1", EIssueFilterType.DISPLAY_FILTERS, { layout: "kanban" }),
    write("M1")
  );
  display.call.answer(problem(404, "planning.module_not_found"));
  await expect(display.promise).rejects.toMatchObject({ problem: { code: "planning.module_not_found" } });
  expect(store.getIssueFilters("M1")?.displayFilters?.layout).toBe("list");
});

it("keeps a confirmed change when an older read answers after it", async () => {
  const { nerve } = tab;
  const initial = modulePreferencesOf();
  const store = await loaded("M1", initial);
  const older = await requestedIssue(nerve, () => store.fetchFilters("ws", "M1"), read("M1"));
  const patch = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "M1", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false }),
    write("M1")
  );
  patch.call.answer(
    json(200, modulePreferencesOf({ display_properties: { ...initial.display_properties, priority: false } }))
  );
  await patch.promise;
  older.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["low"] } })));
  await older.promise;
  expect(store.getIssueFilters("M1")?.displayProperties?.priority).toBe(false);
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(low);
});

it("keeps two modules' preferences apart: their own values, queues and reads", async () => {
  const { nerve } = tab;
  const store = await loaded("M1", modulePreferencesOf());
  await loaded("M2", modulePreferencesOf({ filters: { priority: ["low"] } }));
  expect(store.getIssueFilters("M1")?.richFilters).toEqual({});
  expect(store.getIssueFilters("M2")?.richFilters).toEqual(low);
  const first = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", high), write("M1"));
  const second = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M2", {}), write("M2"));
  second.call.answer(json(200, modulePreferencesOf()));
  await second.promise;
  expect(store.getIssueFilters("M2")?.richFilters).toEqual({});
  expect(store.getIssueFilters("M1")?.richFilters).toEqual({});
  const m2Read = await requestedIssue(nerve, () => store.fetchFilters("ws", "M2"), read("M2"));
  first.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["high"] } })));
  await first.promise;
  m2Read.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["urgent"] } })));
  await m2Read.promise;
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
  expect(store.getIssueFilters("M2")?.richFilters).toEqual({ and: [{ priority__in: "urgent" }] });
});

it("writes no late answer once its generation retires, and sends nothing after", async () => {
  const { app, nerve } = tab;
  const store = await loaded("M1", modulePreferencesOf());
  const late = await requestedIssue(nerve, () => store.fetchFilters("ws", "M2"), read("M2"));
  const change = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", high), write("M1"));
  app.dispose();
  late.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["low"] } })));
  change.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["high"] } })));
  await expect(late.promise).rejects.toBeInstanceOf(SessionChangedError);
  await expect(change.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store.getIssueFilters("M2")).toBeUndefined();
  expect(store.getIssueFilters("M1")?.richFilters).toEqual({});
  await expect(store.updateFilterExpression("ws", "M1", high)).rejects.toBeInstanceOf(SessionChangedError);
  expect(nerve.calls).toHaveLength(3);
});

it("replaces an unread module's filters with one PATCH and seeds the page from nerve's answer", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.moduleIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", high), write("M1"));
  expect(patch.call.body).toEqual({ filters: { priority: ["high"] } });
  const answer = modulePreferencesOf({
    filters: { priority: ["high"] },
    display_filters: { ...modulePreferencesOf().display_filters, layout: "spreadsheet" },
  });
  patch.call.answer(json(200, answer));
  await patch.promise;
  expect(nerve.calls).toHaveLength(1);
  expect(store.getIssueFilters("M1")).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "spreadsheet" },
    displayProperties: answer.display_properties,
  });
});

it("keeps an unread module's replaced filters when a read sent after the change answers older", async () => {
  const { nerve } = tab;
  const store = tab.app.issue.moduleIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", high), write("M1"));
  const later = await requestedIssue(nerve, () => store.fetchFilters("ws", "M1"), read("M1"));
  patch.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["high"] } })));
  await patch.promise;
  const properties = { ...modulePreferencesOf().display_properties, state: false };
  later.call.answer(json(200, modulePreferencesOf({ display_properties: properties })));
  await later.promise;
  expect(nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(high);
  expect(store.getIssueFilters("M1")?.displayProperties).toEqual(properties);
});

it("follows no route: each call names its module and what it holds stays whatever the route", async () => {
  const { app, nerve } = tab;
  const store = app.issue.moduleIssuesFilter;
  runInAction(() => app.router.setQuery({ workspaceSlug: "ws", projectId: "A", moduleId: "M2" }));
  await loaded("M1", modulePreferencesOf({ filters: { priority: ["high"] } }));
  expect(store.getIssueFilters("M2")).toBeUndefined();
  const shown = store.getIssueFilters("M1");
  expect(shown?.richFilters).toEqual(high);
  runInAction(() => app.router.setQuery({ workspaceSlug: "ws", projectId: "B", moduleId: "M1" }));
  expect(store.getIssueFilters("M1")).toEqual(shown);
  expect(store.getIssueFilters("M2")).toBeUndefined();
  runInAction(() => app.router.setQuery({ workspaceSlug: "other", moduleId: "M3" }));
  expect(store.getIssueFilters("M1")).toEqual(shown);
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "M1", low), write("M1"));
  patch.call.answer(json(200, modulePreferencesOf({ filters: { priority: ["low"] } })));
  await patch.promise;
  expect(store.getIssueFilters("M1")?.richFilters).toEqual(low);
  expect(store.getIssueFilters("M3")).toBeUndefined();
  expect("issueFilters" in store).toBe(false);
});

it("keeps group collapse locally under the module page's store type, apart from the cycle page's", async () => {
  const { app, nerve } = tab;
  const alice = accountOf("alice");
  const entry = (key: string, group: string) => ({
    key,
    workspaceSlug: "ws",
    viewId: "M1",
    userId: alice.id,
    filters: { kanban_filters: { group_by: [group], sub_group_by: [] } },
  });
  const data = new Map([["issue_local_filters", JSON.stringify([entry("CYCLE", "other"), entry("MODULE", "mine")])]]);
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => data.set(key, value),
    },
  });
  await seedAccount(nerve, app.user, alice);
  const store = await loaded("M1", modulePreferencesOf());
  expect(store.getIssueFilters("M1")?.kanbanFilters?.group_by).toEqual(["mine"]);
  await store.updateFilters("ws", "M1", EIssueFilterType.KANBAN_FILTERS, { group_by: ["changed"], sub_group_by: [] });
  expect(JSON.parse(data.get("issue_local_filters") ?? "[]")).toEqual([
    entry("CYCLE", "other"),
    { ...entry("MODULE", "mine"), filters: { kanban_filters: { group_by: ["changed"], sub_group_by: [] } } },
  ]);
});
