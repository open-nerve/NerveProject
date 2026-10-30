/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { EIssueFilterType } from "@nerve/constants";
import { accountOf } from "@/store/member/fake-users";
import { issueStoreTab, issuePageOf, requestedIssue } from "../fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "profile" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
const data = new Map<string, string>();
const localStorage = {
  getItem: (key: string) => data.get(key) ?? null,
  setItem: (key: string, value: string) => data.set(key, value),
};
const tabs: Awaited<ReturnType<typeof issueStoreTab>>[] = [];
async function profileTab(account = "alice") {
  const tab = await issueStoreTab();
  tabs.push(tab);
  runInAction(() => {
    tab.app.user.data = { ...accountOf(account), id: account };
  });
  return tab;
}
beforeEach(() => {
  data.clear();
  vi.useFakeTimers();
  vi.stubGlobal("window", { localStorage });
  vi.stubGlobal("localStorage", localStorage);
});
afterEach(() => {
  for (const tab of tabs.splice(0)) tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it("profileViewsKeepIndependentSettings across target users and accounts", async () => {
  const store = (await profileTab()).app.issue.profileIssuesFilter;
  await store.updateFilterExpression("ws", "target", { and: [{ priority__in: "high" }] });
  await store.updateFilterExpression("ws", "other", { and: [{ priority__in: "low" }] });
  const fresh = (await profileTab()).app.issue.profileIssuesFilter;
  await fresh.fetchFilters("ws", "target");
  await fresh.fetchFilters("ws", "other");
  expect(fresh.getIssueFilters("target")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  expect(fresh.getIssueFilters("other")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  const other = (await profileTab("bob")).app.issue.profileIssuesFilter;
  await other.fetchFilters("ws", "target");
  expect(other.getIssueFilters("target")?.richFilters).toEqual({});
});
it("cleans invalid stored fields while retaining a valid member project restriction", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "PROFILE",
        workspaceSlug: "ws",
        viewId: "target",
        userId: "alice",
        filters: {
          rich_filters: {
            and: [{ priority__in: "broken" }, { project_id__in: "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA" }],
          },
        },
      },
    ])
  );
  const store = (await profileTab()).app.issue.profileIssuesFilter;
  await store.fetchFilters("ws", "target");
  expect(store.getIssueFilters("target")?.richFilters).toEqual({
    and: [{ project_id__in: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" }],
  });
});

it("stored project restriction reaches the real member query after invalid fields are cleaned", async () => {
  const tab = await profileTab();
  try {
    data.set(
      "issue_local_filters",
      JSON.stringify([
        {
          key: "PROFILE",
          workspaceSlug: "ws",
          viewId: "target",
          userId: "alice",
          filters: {
            rich_filters: {
              and: [{ priority__in: "bad" }, { project_id__in: "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA" }],
            },
          },
        },
      ])
    );
    await tab.app.issue.profileIssuesFilter.fetchFilters("ws", "target");
    const list = await requestedIssue(
      tab.nerve,
      () => tab.app.issue.profileIssues.fetchIssues("ws", "target", "init-loader", { limit: 20 }, "assigned"),
      ["GET", "/api/v0/workspaces/ws/members/target/issues"]
    );
    expect(list.call.query).toMatchObject({ project_id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" });
    expect(list.call.query).not.toHaveProperty("priority");
    list.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
    await list.promise;
  } finally {
    tab.app.dispose();
  }
});

it("defaults malformed display settings without discarding valid siblings", async () => {
  data.set(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "PROFILE",
        workspaceSlug: "ws",
        viewId: "target",
        userId: "alice",
        filters: {
          display_filters: {
            layout: "broken",
            order_by: 17,
            group_by: { bad: true },
            sub_group_by: "labels",
            show_empty_groups: "false",
            sub_issue: "false",
            calendar: { layout: "week", show_weekends: "false" },
          },
          display_properties: { priority: "false", labels: { bad: true }, assignee: false, state: false },
        },
      },
    ])
  );
  const store = (await profileTab()).app.issue.profileIssuesFilter;
  await store.fetchFilters("ws", "target");
  expect(store.getIssueFilters("target")?.displayFilters).toMatchObject({
    layout: "list",
    order_by: "sort_order",
    group_by: null,
    sub_group_by: null,
    show_empty_groups: false,
    sub_issue: false,
    calendar: { layout: "week", show_weekends: false },
  });
  expect(store.getIssueFilters("target")?.displayProperties).toMatchObject({
    priority: true,
    labels: true,
    assignee: false,
    state: false,
  });
});

it("clears member-unsupported conditions on read and save while retaining project and priority", async () => {
  const id = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
  const input = {
    and: [
      { state_id__in: id },
      { assignee_id__in: id },
      { subscriber_id__in: id },
      { priority__in: "high" },
      { project_id__in: id },
    ],
  };
  data.set(
    "issue_local_filters",
    JSON.stringify([
      { key: "PROFILE", workspaceSlug: "ws", viewId: "target", userId: "alice", filters: { rich_filters: input } },
    ])
  );
  const store = (await profileTab()).app.issue.profileIssuesFilter;
  await store.fetchFilters("ws", "target");
  const expected = { and: [{ priority__in: "high" }, { project_id__in: id }] };
  expect(store.getIssueFilters("target")?.richFilters).toEqual(expected);
  await store.updateFilterExpression("ws", "target", input);
  expect(store.getIssueFilters("target")?.richFilters).toEqual(expected);
  expect(JSON.parse(data.get("issue_local_filters")!)[0].filters.rich_filters).toEqual(expected);
});

it("returns the list read refusal without resetting saved session settings when storage is unavailable", async () => {
  const tab = await profileTab();
  const store = tab.app.issue.profileIssuesFilter;
  await store.fetchFilters("ws", "target");
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.profileIssues.fetchIssues("ws", "target", "init-loader", { limit: 20 }, "created"),
    ["GET", "/api/v0/workspaces/ws/members/target/issues"]
  );
  first.call.answer(json(200, issuePageOf([])));
  await first.promise;
  vi.spyOn(localStorage, "setItem").mockImplementation(() => {
    throw new Error("blocked");
  });
  const next = await requestedIssue(
    tab.nerve,
    () => store.updateFilters("ws", undefined, EIssueFilterType.DISPLAY_FILTERS, { order_by: "-updated_at" }, "target"),
    ["GET", "/api/v0/workspaces/ws/members/target/issues"]
  );
  const rejected = expect(next.promise).rejects.toMatchObject({ status: 503 });
  next.call.answer(problem(503, "unavailable"));
  await rejected;
  expect(store.getIssueFilters("target")?.displayFilters?.order_by).toBe("-updated_at");
  await store.fetchFilters("ws", "target");
  expect(store.getIssueFilters("target")?.displayFilters?.order_by).toBe("-updated_at");
});

it.each(["calendar", "spreadsheet"])(
  "normalizes unsupported profile layout %s without losing valid siblings",
  async (layout) => {
    data.set(
      "issue_local_filters",
      JSON.stringify([
        {
          key: "PROFILE",
          workspaceSlug: "ws",
          viewId: "target",
          userId: "alice",
          filters: {
            display_filters: {
              layout,
              group_by: "priority",
              sub_group_by: "labels",
              sub_issue: false,
              show_empty_groups: false,
              calendar: { layout: "week", show_weekends: false },
            },
            display_properties: { state: false },
          },
        },
      ])
    );
    const store = (await profileTab()).app.issue.profileIssuesFilter;
    await store.fetchFilters("ws", "target");
    expect(store.getIssueFilters("target")?.displayFilters).toMatchObject({
      layout: "list",
      group_by: "priority",
      sub_group_by: null,
      sub_issue: false,
      show_empty_groups: false,
      calendar: { layout: "week", show_weekends: false },
    });
    expect(store.getIssueFilters("target")?.displayProperties?.state).toBe(false);
  }
);
it("retains separate memory preferences for the same target across workspaces", async () => {
  const tab = await profileTab();
  const other = workspaceOf("other");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  const store = tab.app.issue.profileIssuesFilter;
  await store.fetchFilters("ws", "target");
  await store.updateFilterExpression("ws", "target", { priority__in: "high" });
  await store.fetchFilters("other", "target");
  await store.updateFilterExpression("other", "target", { priority__in: "low" });
  expect(store.getIssueFilters("target", "ws")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  expect(store.getIssueFilters("target", "other")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.profileIssues.fetchIssues("ws", "target", "init-loader", { limit: 20 }, "subscribed"),
    ["GET", "/api/v0/workspaces/ws/members/target/issues"]
  );
  expect(first.call.query).toMatchObject({ priority: "high", relation: "subscribed" });
  first.call.answer(json(200, issuePageOf([])));
  await first.promise;
});
