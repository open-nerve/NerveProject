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
  useIssueListMount(type, tab.app.issue, fetchIssues, { limit: 20 }, "assigned");
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

it("does not accept a page that crossed an observed loss and restoration of project membership", async () => {
  const { loadProjects } = await import("@/store/project/fake-projects");
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  const render = () =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    );
  try {
    await act(async () => render());
    await until(() => tab.nerve.calls.length === 1, "old request");
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    f.role = undefined;
    await act(async () => render());
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    f.role = 15;
    await act(async () => render());
    await act(async () => {
      tab.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("old")])));
    });
    await vi.advanceTimersByTimeAsync(100);
    expect.soft(tab.app.issue.issues.held("old")).toBeUndefined();
    expect.soft(tab.app.issue.projectIssues.groups).toBeUndefined();
  } finally {
    await act(async () => root.unmount());
    tab.app.dispose();
  }
});

it("starts a new list before the invalidated request finishes and never reveals the old value", async () => {
  const { loadProjects } = await import("@/store/project/fake-projects");
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  const render = () =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    );
  try {
    await act(async () => render());
    await until(() => tab.nerve.to("/api/v0/projects/A/issues").length === 1, "initial page");
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    f.role = undefined;
    await act(async () => render());
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    f.role = 15;
    await act(async () => render());
    await vi.advanceTimersByTimeAsync(100);
    const pages = tab.nerve.to("/api/v0/projects/A/issues");
    expect.soft(pages).toHaveLength(2);
    expect.soft(tab.app.issue.projectIssues.groups).toBeUndefined();
    await act(async () => {
      pages[0]!.answer(json(200, issuePageOf([issueOf("old")])));
    });
    await vi.advanceTimersByTimeAsync(100);
    expect.soft(tab.app.issue.issues.held("old")).toBeUndefined();
    if (pages[1]) expect.soft(tab.app.issue.projectIssues.getIssueLoader()).toBe("init-loader");
    if (pages[1]) {
      await act(async () => {
        pages[1]!.answer(json(200, issuePageOf([issueOf("new")])));
      });
      await until(() => tab.app.issue.issues.held("new") !== undefined, "fresh page");
    }
    if (pages[1]) expect.soft(tab.app.issue.projectIssues.getIssueIds()).toEqual(["new"]);
  } finally {
    await act(async () => root.unmount());
    tab.app.dispose();
  }
});

it("unmounting one shared consumer does not cancel the other's in-flight request", async () => {
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  const render = (both: boolean) =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        {both && <Layout key="one" tab={tab} type={EIssuesStoreType.PROJECT} />}
        <Layout key="two" tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    );
  try {
    await act(async () => render(true));
    await until(() => tab.nerve.calls.length === 1, "shared page");
    await act(async () => render(false));
    expect(tab.nerve.calls).toHaveLength(1);
    await act(async () => {
      tab.nerve.calls[0]!.answer(json(200, issuePageOf([issueOf("visible")])));
    });
    await until(() => tab.app.issue.issues.held("visible") !== undefined, "shared page answer");
    expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["visible"]);
  } finally {
    await act(async () => root.unmount());
    tab.app.dispose();
  }
});

it("a fresh page winning before the invalidated old page stays visible", async () => {
  const { loadProjects } = await import("@/store/project/fake-projects");
  const tab = await issueStoreTab();
  const root = createRoot(document.createElement("div"));
  const cache = new Map();
  const render = () =>
    root.render(
      <SWRConfig value={{ provider: () => cache }}>
        <Layout tab={tab} type={EIssuesStoreType.PROJECT} />
      </SWRConfig>
    );
  try {
    await act(async () => render());
    await until(() => tab.nerve.to("/api/v0/projects/A/issues").length === 1, "old page");
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    f.role = undefined;
    await act(async () => render());
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    f.role = 15;
    await act(async () => render());
    await until(() => tab.nerve.to("/api/v0/projects/A/issues").length === 2, "new page");
    const pages = tab.nerve.to("/api/v0/projects/A/issues");
    await act(async () => {
      pages[1]!.answer(json(200, issuePageOf([issueOf("new")])));
    });
    await until(() => tab.app.issue.projectIssues.getIssueIds()?.includes("new") === true, "new answer");
    await act(async () => {
      pages[0]!.answer(json(200, issuePageOf([issueOf("old")])));
    });
    await vi.advanceTimersByTimeAsync(100);
    expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["new"]);
    expect(tab.app.issue.issues.held("old")).toBeUndefined();
  } finally {
    await act(async () => root.unmount());
    tab.app.dispose();
  }
});
