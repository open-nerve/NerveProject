/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { EIssueFilterType } from "@nerve/constants";
import { accountOf } from "@/store/member/fake-users";
import { issueStoreTab, issuePageOf, requestedIssue } from "../fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { EIssuesStoreType } from "@nerve/types";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "archive" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
const tabs: Awaited<ReturnType<typeof issueStoreTab>>[] = [];
async function archiveTab(account = "alice") {
  const tab = await issueStoreTab();
  tabs.push(tab);
  runInAction(() => {
    tab.app.user.data = { ...accountOf(account), id: account };
  });
  return tab;
}

const data = new Map<string, string>();
const localStorage = {
  getItem: (key: string) => data.get(key) ?? null,
  setItem: (key: string, value: string) => data.set(key, value),
};
beforeEach(() => {
  data.clear();
  vi.useFakeTimers();
  vi.stubGlobal("window", { localStorage });
});
afterEach(() => {
  for (const tab of tabs.splice(0)) tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("archiveFiltersSurviveFreshStore and isolate accounts", async () => {
  const first = (await archiveTab()).app.issue.archivedIssuesFilter;
  await first.updateFilterExpression("ws", "A", { and: [{ priority__in: "high" }] });
  const fresh = (await archiveTab()).app.issue.archivedIssuesFilter;
  await fresh.fetchFilters("ws", "A");
  expect(fresh.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  const other = (await archiveTab("bob")).app.issue.archivedIssuesFilter;
  await other.fetchFilters("ws", "A");
  expect(other.getIssueFilters("A")?.richFilters).toEqual({});
});

it("legacyArchiveKeyMigratesOnce only for its known account", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "A",
        userId: "alice",
        filters: { richFilters: { and: [{ priority__in: "low" }] } },
      },
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "other",
        filters: { richFilters: { priority__in: "high" } },
      },
    ])
  );
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  const entries = JSON.parse(data.get("issue_local_filters")!);
  expect(entries[0].filters.richFilters).toBeUndefined();
  expect(entries[0].filters.rich_filters).toEqual({ and: [{ priority__in: "low" }] });
  await store.fetchFilters("ws", "other");
  expect(store.getIssueFilters("other")?.richFilters).toEqual({});
});

it.each(["{", "{}", '[null, 1, {"filters":null}]'])("invalidStorageDefaultsWithoutLeakingScope: %s", async (value) => {
  data.set("issue_local_filters", value);
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.richFilters).toEqual({});
});

it("uses defaults and accepts updates when browser storage throws", async () => {
  vi.stubGlobal("window", {
    localStorage: {
      getItem: () => {
        throw new Error("blocked");
      },
      setItem: () => {
        throw new Error("blocked");
      },
    },
  });
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await expect(store.fetchFilters("ws", "A")).resolves.toBeUndefined();
  await expect(store.updateFilterExpression("ws", "A", { priority__in: "high" })).resolves.toBeUndefined();
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
});

it.each(["blocked reads", "stale stored values"])(
  "keeps session archive settings when returning to the page with %s",
  async (failure) => {
    const store = (await archiveTab()).app.issue.archivedIssuesFilter;
    await store.fetchFilters("ws", "A");
    await store.updateFilterExpression("ws", "A", { priority__in: "low" });
    if (failure === "blocked reads")
      vi.spyOn(localStorage, "getItem").mockImplementation(() => {
        throw new Error("blocked");
      });
    vi.spyOn(localStorage, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });

    await store.updateFilterExpression("ws", "A", { priority__in: "high" });
    await store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_FILTERS, {
      order_by: "-priority",
      group_by: "state",
      sub_issue: false,
    });
    await store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { state: false });
    await store.updateFilters("ws", "A", EIssueFilterType.KANBAN_FILTERS, { group_by: ["done"], sub_group_by: [] });

    // The route's session SWR revalidates local settings on every mount.
    await store.fetchFilters("ws", "B");
    await store.fetchFilters("ws", "A");
    expect(store.getIssueFilters("A")).toMatchObject({
      richFilters: { and: [{ priority__in: "high" }] },
      displayFilters: { order_by: "-priority", group_by: "state", sub_issue: false },
      displayProperties: { state: false },
      kanbanFilters: { group_by: ["done"], sub_group_by: [] },
    });
    expect(store.getIssueFilters("B")?.richFilters).toEqual({});
  }
);

it("writing an existing legacy entry removes the obsolete key", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "A",
        userId: "alice",
        filters: { richFilters: { priority__in: "low" } },
      },
    ])
  );
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.updateFilterExpression("ws", "A", { priority__in: "high" });
  const filters = JSON.parse(data.get("issue_local_filters")!)[0].filters;
  expect(filters.richFilters).toBeUndefined();
  expect(filters.rich_filters).toEqual({ and: [{ priority__in: "high" }] });
});

it("defaults malformed nested kanban data before consumers use it", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "A",
        userId: "alice",
        filters: { kanban_filters: { group_by: "broken", sub_group_by: [null] } },
      },
    ])
  );
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.kanbanFilters).toEqual({ group_by: [], sub_group_by: [] });
});

it("defaults malformed display fields while preserving valid values and false", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "A",
        userId: "alice",
        filters: {
          display_filters: {
            layout: "broken",
            order_by: "broken",
            group_by: { bad: true },
            sub_group_by: "priority",
            show_empty_groups: "false",
            sub_issue: false,
            calendar: { layout: "broken", show_weekends: false },
          },
          display_properties: { priority: "false", labels: { bad: true }, state: false, key: false },
        },
      },
    ])
  );
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({
    layout: "list",
    order_by: "sort_order",
    group_by: null,
    sub_group_by: null,
    show_empty_groups: false,
    sub_issue: false,
    calendar: { layout: "month", show_weekends: false },
  });
  expect(store.getIssueFilters("A")?.displayProperties).toMatchObject({
    priority: true,
    labels: true,
    state: false,
    key: false,
  });
});

it("normalizes archive layout while retaining valid sort, group and booleans when calendar is malformed", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "A",
        userId: "alice",
        filters: {
          display_filters: {
            layout: "kanban",
            order_by: "-priority",
            group_by: "state",
            sub_group_by: null,
            show_empty_groups: false,
            sub_issue: false,
            calendar: [],
          },
          display_properties: { priority: false },
        },
      },
    ])
  );
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({
    layout: "list",
    order_by: "-priority",
    group_by: "state",
    sub_group_by: null,
    show_empty_groups: false,
    sub_issue: false,
    calendar: { layout: "month", show_weekends: false },
  });
  expect(store.getIssueFilters("A")?.displayProperties?.priority).toBe(false);
});

it("returns a real list refusal without rolling back locally saved values when storage is blocked", async () => {
  const tab = await archiveTab();
  const store = tab.app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  first.call.answer(json(200, issuePageOf([])));
  await first.promise;
  vi.spyOn(localStorage, "setItem").mockImplementation(() => {
    throw new Error("blocked");
  });
  const sent = await requestedIssue(
    tab.nerve,
    () => store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_FILTERS, { order_by: "-priority", sub_issue: false }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(sent.result.settled).toBe(false);
  sent.call.answer(problem(503, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 503, problem: { code: "unavailable" } });
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({ order_by: "-priority", sub_issue: false });
  expect(tab.app.issue.archivedIssues.currentList?.error).toBeDefined();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("restores independent workspace/project values and always selects archived rows", async () => {
  const tab = await archiveTab();
  const store = tab.app.issue.archivedIssuesFilter;
  await store.updateFilterExpression("ws", "A", { priority__in: "high" });
  await store.updateFilterExpression("other", "A", { priority__in: "low" });
  await store.updateFilterExpression("ws", "B", { priority__in: "medium" });
  await store.fetchFilters("ws", "A");
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  const first = await requestedIssue(
    tab.nerve,
    () =>
      tab.app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", { limit: 20, query: { archived: "false" } }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  first.call.answer(json(200, issuePageOf([])));
  await first.promise;
  expect(first.call.query).toMatchObject({ archived: "true", priority: "high" });
  await store.fetchFilters("other", "A");
  expect(store.getIssueFilters("A", "other")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  await store.fetchFilters("ws", "B");
  expect(store.getIssueFilters("B")?.richFilters).toEqual({ and: [{ priority__in: "medium" }] });
});
it("normalizes updates to archive capabilities without losing group collapse or false", async () => {
  const store = (await archiveTab()).app.issue.archivedIssuesFilter;
  await store.fetchFilters("ws", "A");
  await store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_FILTERS, {
    layout: "kanban",
    group_by: "state",
    sub_group_by: "priority",
    show_empty_groups: false,
  });
  await store.updateFilters("ws", "A", EIssueFilterType.KANBAN_FILTERS, {
    group_by: ["state"],
    sub_group_by: ["high"],
  });
  const again = (await archiveTab()).app.issue.archivedIssuesFilter;
  await again.fetchFilters("ws", "A");
  expect(again.getIssueFilters("A")?.displayFilters).toMatchObject({
    layout: "list",
    group_by: "state",
    sub_group_by: null,
    show_empty_groups: false,
  });
  expect(again.getIssueFilters("A")?.kanbanFilters).toEqual({ group_by: ["state"], sub_group_by: [] });
});
it("does not let a retired root overwrite local settings", async () => {
  const tab = await archiveTab();
  const store = tab.app.issue.archivedIssuesFilter;
  await store.updateFilterExpression("ws", "A", { priority__in: "high" });
  const before = data.get("issue_local_filters");
  tab.app.dispose();
  await expect(store.updateFilterExpression("ws", "A", { priority__in: "low" })).rejects.toBeInstanceOf(
    SessionChangedError
  );
  expect(data.get("issue_local_filters")).toBe(before);
});
