/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's view's page mounts its list (M7/P6b spec 2.3, point 10; re-97 m8): each page of the route's
// globalViewId its own list, two default views never one; for any role of the workspace, its guests too; and over the
// projects the page's filters name, so that a change of another project's role does not read the page again, where a
// page of every project does.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { sessionKey } from "@/lib/session-key";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { useIssueListMount } from "./use-issue-list-mount";

const f = vi.hoisted(() => ({ params: {} as Record<string, string>, role: undefined as number | undefined }));
const keys: unknown[] = [];
vi.mock("react-router", () => ({ useParams: () => f.params }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => undefined,
    getWorkspaceRoleByWorkspaceSlug: () => f.role,
  }),
}));
vi.mock("@/lib/use-session-swr", () => ({
  useSessionSWR: (key: unknown) => {
    keys.push(key);
    return {};
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  f.role = 15;
  keys.length = 0;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const fetch = (() => Promise.resolve(undefined)) as never;
/** The list's mount key on the route of the page given, as the session key gives it; null for none. */
function keyOf(globalViewId: string) {
  f.params = { workspaceSlug: "ws", globalViewId };
  useIssueListMount(EIssuesStoreType.GLOBAL, tab.app.issue, fetch, { limit: 100 });
  const key = keys.at(-1);
  return key ? sessionKey({ status: "signed-in", loginId: "x" }, ...(key as [string, ...string[]])) : null;
}

it("mounts each page's own list, the route's globalViewId in its key", () => {
  const all = keyOf("all-issues");
  const assigned = keyOf("assigned");
  expect(all).toContain("all-issues");
  expect(assigned).toContain("assigned");
  expect(all).not.toEqual(assigned);
});

it.each([
  [undefined, false],
  [5, true],
  [15, true],
  [20, true],
])("with the workspace role %s mounts a list: %s", (role, mounted) => {
  f.role = role;
  expect(keyOf("all-issues") !== null).toBe(mounted);
});

const P = "019a0000-0000-7000-8000-0000000000b1";
const Q = "019a0000-0000-7000-8000-0000000000b2";

it("reads the page again on a change of a project its filters name, never of another one", async () => {
  const named = projectOf("P", tab.workspace.id, { id: P });
  const other = projectOf("Q", tab.workspace.id, { id: Q });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [named, other]);
  const page = workspacePageId("ws", "all-issues");
  await tab.app.issue.globalIssuesFilter.fetchFilters("ws", page);
  await tab.app.issue.globalIssuesFilter.updateFilterExpression("ws", page, { and: [{ project_id__in: P }] });
  const before = keyOf("all-issues");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [named, { ...other, member_role: 5 }]);
  expect(keyOf("all-issues"), "another project's role").toEqual(before);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...named, member_role: 5 },
    { ...other, member_role: 5 },
  ]);
  expect(keyOf("all-issues"), "the page's project's role").not.toEqual(before);
});
