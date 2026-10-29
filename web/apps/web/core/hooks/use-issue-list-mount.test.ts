/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { sessionKey } from "@/lib/session-key";
import { issueOf, issuePageOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { useIssueListMount } from "./use-issue-list-mount";

const f = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  role: undefined as number | undefined,
  login: "x",
  calls: [] as unknown[][],
}));
vi.mock("react-router", () => ({ useParams: () => f.params }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => f.role,
    getWorkspaceRoleByWorkspaceSlug: () => f.role,
  }),
}));
vi.mock("@/lib/use-session-swr", () => ({
  useSessionSWR: (key: unknown, fetch: unknown) => {
    f.calls.push([key, fetch]);
    return {};
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
beforeEach(() => {
  vi.useFakeTimers();
  f.params = { workspaceSlug: "ws", projectId: "A", userId: "actor" };
  f.role = undefined;
  f.calls = [];
});

afterEach(() => vi.useRealTimers());

it("gates actual list fetches for unknown, nonmember, and guest scopes", async () => {
  const tab = await issueStoreTab();
  const project = (_loader: never, options: never) =>
    tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", options);
  const archived = (_loader: never, options: never) =>
    tab.app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", options);
  const profile = (_loader: never, options: never) =>
    tab.app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", options, "assigned");
  const fetchers = [project, archived, profile] as never[];
  const types = [EIssuesStoreType.PROJECT, EIssuesStoreType.ARCHIVED, EIssuesStoreType.PROFILE];
  for (const role of [undefined, 5, 15, 20]) {
    f.role = role;
    f.calls = [];
    types.forEach((type, i) => useIssueListMount(type, fetchers[i]!, { limit: 20 }, "assigned"));
    expect(f.calls.map(([key]) => !!key)).toEqual(
      role === undefined ? [false, false, false] : role === 5 ? [true, false, false] : [true, true, true]
    );
  }
  f.params.projectId = undefined as never;
  f.role = 15;
  f.calls = [];
  useIssueListMount(EIssuesStoreType.PROJECT, project as never, { limit: 20 });
  const [key] = f.calls[0]!;
  expect(sessionKey({ status: "signed-in", loginId: "x" }, ...(key as [string, ...string[]]))).toBeNull();
  expect(tab.nerve.calls).toEqual([]);
  tab.app.dispose();
});

it("keys a real list by login and drops a retired session's late answer; refusal reaches the caller", async () => {
  const x = await issueStoreTab();
  const y = await issueStoreTab();
  f.role = 15;
  const run = (tab: typeof x) => {
    f.calls = [];
    useIssueListMount(
      EIssuesStoreType.PROJECT,
      ((_loader: never, options: never) =>
        tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", options)) as never,
      { limit: 20 }
    );
    return f.calls[0]!;
  };
  const [kx, fetchX] = run(x);
  const [ky, fetchY] = run(y);
  expect(sessionKey({ status: "signed-in", loginId: "x" }, ...(kx as [string, ...string[]]))).not.toEqual(
    sessionKey({ status: "signed-in", loginId: "y" }, ...(ky as [string, ...string[]]))
  );
  const old = track((fetchX as () => Promise<unknown>)());
  await until(() => x.nerve.calls.length === 1, "old list");
  x.app.dispose();
  x.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("old")])));
  await until(() => old.settled, "old answer");
  expect(old.value).toBeUndefined();
  expect(x.app.issue.issues.held("old")).toBeUndefined();
  expect(y.app.issue.issues.held("old")).toBeUndefined();
  const fresh = track((fetchY as () => Promise<unknown>)());
  await until(() => y.nerve.calls.length === 1, "new list");
  y.nerve.calls[0]!.answer(problem(403, "forbidden"));
  await until(() => fresh.settled, "refusal");
  expect(fresh.error).toBeDefined();
  expect(y.app.issue.projectIssues.currentList?.error).toBeDefined();
  y.app.dispose();
});
