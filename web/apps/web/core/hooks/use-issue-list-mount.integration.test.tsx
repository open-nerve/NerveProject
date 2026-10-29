// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { issueOf, issuePageOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { useIssueListMount } from "./use-issue-list-mount";
import type { IssueActions } from "./use-issues-actions";

const f = vi.hoisted(() => ({
  login: "x",
  role: undefined as number | undefined,
  params: { workspaceSlug: "ws", projectId: "A", userId: "actor" },
}));
vi.mock("react-router", () => ({ useParams: () => f.params }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: f.login }) }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => f.role,
    getWorkspaceRoleByWorkspaceSlug: () => f.role,
  }),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  f.login = "x";
  f.role = 15;
});
afterEach(() => vi.useRealTimers());

type Tab = Awaited<ReturnType<typeof issueStoreTab>>;
function Layout({ tab, type }: { tab: Tab; type: EIssuesStoreType }) {
  const fetchIssues: IssueActions["fetchIssues"] = (loader, options) =>
    type === EIssuesStoreType.PROFILE
      ? tab.app.issue.profileIssues.fetchIssues("ws", "actor", loader, options, "assigned")
      : type === EIssuesStoreType.ARCHIVED
        ? tab.app.issue.archivedIssues.fetchIssues("ws", "A", loader, options)
        : tab.app.issue.projectIssues.fetchIssues("ws", "A", loader, options);
  useIssueListMount(type, fetchIssues, { limit: 20 }, "assigned");
  return null;
}

it("mounts the list under each login and discards the prior session's late HTTP answer", async () => {
  const x = await issueStoreTab();
  const y = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  const render = (tab: Tab) =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    );
  await act(async () => render(x));
  await until(() => x.nerve.calls.length === 1, "X list");
  expect(x.nerve.calls[0]?.path).toBe("/api/v0/projects/A/issues");
  x.app.dispose();
  f.login = "y";
  await act(async () => render(y));
  await until(() => y.nerve.calls.length === 1, "Y list");
  await act(async () => {
    x.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("old")])));
    y.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("new")])));
  });
  await until(() => y.app.issue.issues.held("new") !== undefined, "Y list answer");
  expect(x.app.issue.issues.held("old")).toBeUndefined();
  expect(y.app.issue.issues.held("old")).toBeUndefined();
  await act(async () => root.unmount());
  y.app.dispose();
});

it("mounting guest archive and profile, or an unknown project, sends no list GET", async () => {
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  f.role = 5;
  await act(async () =>
    root.render(
      <>
        <Layout tab={tab} type={EIssuesStoreType.ARCHIVED} />
        <Layout tab={tab} type={EIssuesStoreType.PROFILE} />
      </>
    )
  );
  expect(tab.nerve.calls).toEqual([]);
  f.role = undefined;
  await act(async () => root.render(<Layout tab={tab} type={EIssuesStoreType.PROJECT} />));
  expect(tab.nerve.calls).toEqual([]);
  await act(async () => root.unmount());
  tab.app.dispose();
});

it("drops the old page when project membership disappears in the same login", async () => {
  const { loadProjects } = await import("@/store/project/fake-projects");
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  await act(async () => root.render(<Layout tab={tab} type={EIssuesStoreType.PROJECT} />));
  await until(() => tab.nerve.calls.length === 1, "project page");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  f.role = undefined;
  await act(async () => root.render(<Layout tab={tab} type={EIssuesStoreType.PROJECT} />));
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("old")])));
  });
  await vi.advanceTimersByTimeAsync(100);
  expect(tab.app.issue.issues.held("old")).toBeUndefined();
  expect(tab.app.issue.projectIssues.groups).toBeUndefined();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  f.role = 15;
  const prior = tab.nerve.calls.length;
  await act(async () => root.render(<Layout tab={tab} type={EIssuesStoreType.PROJECT} />));
  await until(() => tab.nerve.calls.length > prior, "restored membership list");
  await act(async () => {
    tab.nerve.calls.at(-1)!.answer(json(200, issuePageOf([issueOf("restored")])));
  });
  await until(() => tab.app.issue.issues.held("restored") !== undefined, "restored list answer");
  expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["restored"]);
  await act(async () => root.unmount());
  tab.app.dispose();
});

it("guest project list is sent without inventing an own-only filter", async () => {
  const { loadProjects, projectOf } = await import("@/store/project/fake-projects");
  const tab = await issueStoreTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    projectOf("A", tab.workspace.id, { id: "A", member_role: 5 }),
  ]);
  tab.nerve.calls.length = 0;
  f.role = 5;
  const root = createRoot(document.createElement("div"));
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    )
  );
  await until(() => tab.nerve.calls.length === 1, "guest list");
  expect(tab.nerve.calls[0]?.path).toBe("/api/v0/projects/A/issues");
  expect(tab.nerve.calls[0]?.query).not.toHaveProperty("assignee_id");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("server-visible")])));
  });
  await until(() => tab.app.issue.issues.held("server-visible") !== undefined, "guest answer");
  await act(async () => root.unmount());
  tab.app.dispose();
});

it("deduplicates simultaneous mounts while the same list request remains in flight", async () => {
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    )
  );
  await until(() => tab.nerve.calls.length === 1, "shared list");
  await vi.advanceTimersByTimeAsync(50);
  expect(tab.nerve.calls).toHaveLength(1);
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, issuePageOf([])));
  });
  await act(async () => root.unmount());
  tab.app.dispose();
});
