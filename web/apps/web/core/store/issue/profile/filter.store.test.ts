/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { beforeEach, expect, it, vi } from "vitest";
import type { IIssueRootStore } from "../root.store";
import { ProfileIssuesFilter } from "./filter.store";
const data = new Map<string, string>();
const localStorage = {
  getItem: (key: string) => data.get(key) ?? null,
  setItem: (key: string, value: string) => data.set(key, value),
};
const root = (account = "alice") =>
  ({
    currentUserId: account,
    profileIssues: { fetchIssuesWithExistingPagination: vi.fn() },
  }) as unknown as IIssueRootStore;
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
beforeEach(() => {
  data.clear();
  vi.stubGlobal("window", { localStorage });
  vi.stubGlobal("localStorage", localStorage);
});
it("profileViewsKeepIndependentSettings across target users and accounts", async () => {
  const store = new ProfileIssuesFilter(root());
  await store.updateFilterExpression("ws", "target", { and: [{ priority__in: "high" }] });
  await store.updateFilterExpression("ws", "other", { and: [{ priority__in: "low" }] });
  const fresh = new ProfileIssuesFilter(root());
  await fresh.fetchFilters("ws", "target");
  await fresh.fetchFilters("ws", "other");
  expect(fresh.getIssueFilters("target")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  expect(fresh.getIssueFilters("other")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  const other = new ProfileIssuesFilter(root("bob"));
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
  const store = new ProfileIssuesFilter(root());
  await store.fetchFilters("ws", "target");
  expect(store.getIssueFilters("target")?.richFilters).toEqual({
    and: [{ project_id__in: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" }],
  });
});

it("stored project restriction reaches the real member query after invalid fields are cleaned", async () => {
  const { issueStoreTab, requestedIssue } = await import("../fake-issues");
  const { json } = await import("@/lib/auth/fake-nerve");
  const { runInAction } = await import("mobx");
  vi.useFakeTimers();
  const tab = await issueStoreTab();
  try {
    runInAction(() => {
      tab.app.user.data = { id: "alice" } as NonNullable<typeof tab.app.user.data>;
    });
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
  const store = new ProfileIssuesFilter(root());
  await store.fetchFilters("ws", "target");
  expect(store.getIssueFilters("target")?.displayFilters).toMatchObject({
    layout: "list",
    order_by: "sort_order",
    group_by: null,
    sub_group_by: "labels",
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
  const store = new ProfileIssuesFilter(root());
  await store.fetchFilters("ws", "target");
  const expected = { and: [{ priority__in: "high" }, { project_id__in: id }] };
  expect(store.getIssueFilters("target")?.richFilters).toEqual(expected);
  await store.updateFilterExpression("ws", "target", input);
  expect(store.getIssueFilters("target")?.richFilters).toEqual(expected);
  expect(JSON.parse(data.get("issue_local_filters")!)[0].filters.rich_filters).toEqual(expected);
});

it("returns the list read refusal after a filter change", async () => {
  const failure = new Error("list refused");
  let reject!: (error: Error) => void;
  const read = new Promise<void>((_resolve, fail) => {
    reject = fail;
  });
  void read.catch(() => {});
  const owner = root();
  vi.mocked(owner.profileIssues.fetchIssuesWithExistingPagination).mockReturnValue(read as never);
  const changed = new ProfileIssuesFilter(owner).updateFilterExpression("ws", "target", { priority__in: "high" });
  let settled = false;
  void changed.then(
    () => (settled = true),
    () => (settled = true)
  );
  await Promise.resolve();
  expect(settled).toBe(false);
  reject(failure);
  await expect(changed).rejects.toBe(failure);
});
