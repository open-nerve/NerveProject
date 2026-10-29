/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { IIssueRootStore } from "../root.store";
import { ArchivedIssuesFilter } from "./filter.store";

const data = new Map<string, string>();
const localStorage = {
  getItem: (key: string) => data.get(key) ?? null,
  setItem: (key: string, value: string) => data.set(key, value),
};
const root = (account = "alice") =>
  ({
    currentUserId: account,
    archivedIssues: { fetchIssuesWithExistingPagination: vi.fn() },
  }) as unknown as IIssueRootStore;
beforeEach(() => {
  data.clear();
  vi.stubGlobal("window", { localStorage });
});

it("archiveFiltersSurviveFreshStore and isolate accounts", async () => {
  const first = new ArchivedIssuesFilter(root());
  await first.updateFilterExpression("ws", "project", { and: [{ priority__in: "high" }] });
  const fresh = new ArchivedIssuesFilter(root());
  await fresh.fetchFilters("ws", "project");
  expect(fresh.getIssueFilters("project")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  const other = new ArchivedIssuesFilter(root("bob"));
  await other.fetchFilters("ws", "project");
  expect(other.getIssueFilters("project")?.richFilters).toEqual({});
});

it("legacyArchiveKeyMigratesOnce only for its known account", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "project",
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
  const store = new ArchivedIssuesFilter(root());
  await store.fetchFilters("ws", "project");
  expect(store.getIssueFilters("project")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  const entries = JSON.parse(data.get("issue_local_filters")!);
  expect(entries[0].filters.richFilters).toBeUndefined();
  expect(entries[0].filters.rich_filters).toEqual({ and: [{ priority__in: "low" }] });
  await store.fetchFilters("ws", "other");
  expect(store.getIssueFilters("other")?.richFilters).toEqual({});
});

it.each(["{", "{}", '[null, 1, {"filters":null}]'])("invalidStorageDefaultsWithoutLeakingScope: %s", async (value) => {
  data.set("issue_local_filters", value);
  const store = new ArchivedIssuesFilter(root());
  await store.fetchFilters("ws", "project");
  expect(store.getIssueFilters("project")?.richFilters).toEqual({});
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
  const store = new ArchivedIssuesFilter(root());
  await expect(store.fetchFilters("ws", "project")).resolves.toBeUndefined();
  await expect(store.updateFilterExpression("ws", "project", { priority__in: "high" })).resolves.toBeUndefined();
  expect(store.getIssueFilters("project")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
});

it("writing an existing legacy entry removes the obsolete key", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "project",
        userId: "alice",
        filters: { richFilters: { priority__in: "low" } },
      },
    ])
  );
  const store = new ArchivedIssuesFilter(root());
  await store.updateFilterExpression("ws", "project", { priority__in: "high" });
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
        viewId: "project",
        userId: "alice",
        filters: { kanban_filters: { group_by: "broken", sub_group_by: [null] } },
      },
    ])
  );
  const store = new ArchivedIssuesFilter(root());
  await store.fetchFilters("ws", "project");
  expect(store.getIssueFilters("project")?.kanbanFilters).toEqual({ group_by: [], sub_group_by: [] });
});

it("defaults malformed display fields while preserving valid values and false", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "project",
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
  const store = new ArchivedIssuesFilter(root());
  await store.fetchFilters("ws", "project");
  expect(store.getIssueFilters("project")?.displayFilters).toMatchObject({
    layout: "list",
    order_by: "sort_order",
    group_by: null,
    sub_group_by: "priority",
    show_empty_groups: false,
    sub_issue: false,
    calendar: { layout: "month", show_weekends: false },
  });
  expect(store.getIssueFilters("project")?.displayProperties).toMatchObject({
    priority: true,
    labels: true,
    state: false,
    key: false,
  });
});

it("retains valid layout, sort, group and booleans when calendar is malformed", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "ws",
        viewId: "project",
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
  const store = new ArchivedIssuesFilter(root());
  await store.fetchFilters("ws", "project");
  expect(store.getIssueFilters("project")?.displayFilters).toMatchObject({
    layout: "kanban",
    order_by: "-priority",
    group_by: "state",
    sub_group_by: null,
    show_empty_groups: false,
    sub_issue: false,
    calendar: { layout: "month", show_weekends: false },
  });
  expect(store.getIssueFilters("project")?.displayProperties?.priority).toBe(false);
});
