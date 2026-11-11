/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The page-preferences base's own branches, through its project page; the page's shared rules are in
// ../project/filter.store.test.ts.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueFilterType } from "@nerve/constants";
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

const high = { and: [{ priority__in: "high" }] };
const properties = () => preferencesOf().display_properties;

it("replaces a whole field with nothing read: only that field is sent and the answer seeds the page", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "A", high), [
    "PATCH",
    "/api/v0/me/projects/A/preferences",
  ]);
  expect(patch.call.body).toEqual({ filters: { priority: ["high"] } });
  patch.call.answer(
    json(200, preferencesOf({ filters: { priority: ["high"] }, display_properties: { ...properties(), state: false } }))
  );
  await patch.promise;
  expect(store.getIssueFilters("A")).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "list", group_by: null, order_by: "-created_at" },
    displayProperties: { state: false, priority: true },
  });
});

it("keeps a field confirmed with nothing read when a read started before it answers older", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "A", high), [
    "PATCH",
    "/api/v0/me/projects/A/preferences",
  ]);
  patch.call.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  await patch.promise;
  read.call.answer(json(200, preferencesOf({ display_properties: { ...properties(), state: false } })));
  await read.promise;
  expect(store.getIssueFilters("A")?.richFilters).toEqual(high);
  expect(store.getIssueFilters("A")?.displayProperties?.state).toBe(false);
  const list = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(list.call.query).toMatchObject({ priority: "high" });
  list.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await list.promise;
});

it("seeds nothing when a whole-field change with nothing read is refused", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const patch = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "A", high), [
    "PATCH",
    "/api/v0/me/projects/A/preferences",
  ]);
  patch.call.answer(problem(403, "forbidden"));
  await expect(patch.promise).rejects.toBeDefined();
  expect(store.getIssueFilters("A")).toBeUndefined();
});

it("keeps group collapse locally under the page's store type", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const alice = accountOf("alice");
  const entry = (key: string, group: string) => ({
    key,
    workspaceSlug: "ws",
    viewId: "A",
    userId: alice.id,
    filters: { kanban_filters: { group_by: [group], sub_group_by: [] } },
  });
  const data = new Map([
    ["issue_local_filters", JSON.stringify([entry("ARCHIVED", "other"), entry("PROJECT", "mine")])],
  ]);
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => data.set(key, value),
    },
  });
  await seedAccount(nerve, app.user, alice);
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, preferencesOf()));
  await read.promise;
  expect(store.getIssueFilters("A")?.kanbanFilters?.group_by).toEqual(["mine"]);
  await store.updateFilters("ws", "A", EIssueFilterType.KANBAN_FILTERS, { group_by: ["changed"], sub_group_by: [] });
  const saved = JSON.parse(data.get("issue_local_filters") ?? "[]");
  expect(saved).toEqual([
    entry("ARCHIVED", "other"),
    { ...entry("PROJECT", "mine"), filters: { kanban_filters: { group_by: ["changed"], sub_group_by: [] } } },
  ]);
});

it("refuses a partial change of a field with nothing read, without a request", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  await expect(
    store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { state: false })
  ).rejects.toBeDefined();
  expect(nerve.calls).toHaveLength(0);
  expect(store.getIssueFilters("A")).toBeUndefined();
});
