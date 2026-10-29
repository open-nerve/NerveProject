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
      tab.app.issue.currentUserId = "alice";
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
