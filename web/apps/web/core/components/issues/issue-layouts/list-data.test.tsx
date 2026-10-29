// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab, issuePageOf } from "@/store/issue/fake-issues";
import { loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { ProjectLayoutRoot } from "./roots/project-layout-root";
import { ArchivedIssueLayoutRoot } from "./roots/archived-issue-layout-root";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";

const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  role: 15 as number | undefined,
  params: { workspaceSlug: "ws", projectId: "A", userId: "actor", profileViewId: "assigned" },
}));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return {
    StoreContext: createContext(undefined),
    get rootStore() {
      return fixture.app;
    },
  };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "A" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "A" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => fixture.params,
}));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => fixture.role,
    getWorkspaceRoleByWorkspaceSlug: () => fixture.role,
    allowPermissions: () => true,
  }),
  useUser: () => ({ data: { id: "actor" } }),
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.role = 15;
  fixture.params.projectId = "A";
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", userId: "actor" }));
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.useRealTimers();
});
function mount(child = <ProjectLayoutRoot />) {
  view.render(
    <StoreContext.Provider value={tab.app}>
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter>{child}</MemoryRouter>
      </SWRConfig>
    </StoreContext.Provider>
  );
}
it("mountedProjectListUsesGeneratedRequest waits for saved preferences and sends their query", async () => {
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  expect(tab.nerve.calls[0]).toMatchObject({ method: "GET", path: "/api/v0/me/projects/A/preferences" });
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "list"));
  expect(tab.nerve.calls[1]).toMatchObject({
    method: "GET",
    path: "/api/v0/projects/A/issues",
    query: { priority: "high", order_by: "-created_at", sub_issue: "true", limit: "100" },
  });
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});
it.each([
  ["list", "month", "100"],
  ["kanban", "month", "30"],
  ["spreadsheet", "month", "100"],
  ["calendar", "month", "4"],
  ["calendar", "week", "30"],
] as const)("the actual %s layout (%s) mounts its first query", async (layout, calendar, limit) => {
  const prefs = preferencesOf();
  prefs.display_filters.layout = layout;
  prefs.display_filters.calendar.layout = calendar;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, prefs));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "layout list"));
  const query = tab.nerve.calls[1]!.query;
  expect(query).toMatchObject({ order_by: "-created_at", sub_issue: "true", limit });
  if (layout === "calendar") {
    const dates = tab.app.issue.issueCalendarView.getStartAndEndDate(calendar)!;
    expect(query).toMatchObject({ group_by: "target_date", target_date: `${dates.startDate}..${dates.endDate}` });
  }
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});

it("a rejected preference displays an error without a list request", async () => {
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(problem(403, "permission_denied"));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(tab.nerve.calls).toHaveLength(1);
});

it.each([
  ["archived", <ArchivedIssueLayoutRoot key="archived" />],
  ["profile", <ProfileIssuesPage key="profile" type="assigned" />],
] as const)("guest cannot mount %s list", async (_name, child) => {
  fixture.role = 5;
  await act(async () => {
    mount(child);
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toEqual([]);
});
it("a non-member cannot mount a project list", async () => {
  fixture.role = undefined;
  await act(async () => {
    mount();
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toEqual([]);
});
it("a guest project list leaves own-only enforcement to the server", async () => {
  fixture.role = 5;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "guest preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf()));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "guest list"));
  expect(tab.nerve.calls[1]!.query).not.toHaveProperty("assignee_id");
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});
it.each([
  ["archived", <ArchivedIssueLayoutRoot key="archived" />, "/api/v0/projects/A/issues"],
  ["profile", <ProfileIssuesPage key="profile" type="assigned" />, "/api/v0/workspaces/ws/members/actor/issues"],
] as const)("member mounts the %s list", async (_name, child, path) => {
  await act(async () => mount(child));
  await act(async () => until(() => tab.nerve.calls.length === 1, "list"));
  expect(tab.nerve.calls[0]).toMatchObject({
    method: "GET",
    path,
    query: _name === "archived" ? { archived: "true" } : { relation: "assigned" },
  });
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, issuePageOf([])));
  });
});

it("switching projects waits for the new preferences before selecting its list", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  tab.nerve.calls.length = 0;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "A preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "A list"));
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
  fixture.params.projectId = "B";
  await act(async () => {
    runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "B", userId: "actor" }));
    mount();
  });
  await act(async () => until(() => tab.nerve.calls.length === 3, "B preferences"));
  expect(tab.nerve.calls[2]).toMatchObject({ path: "/api/v0/me/projects/B/preferences" });
  await act(async () => {
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 4, "B list"));
  expect(tab.nerve.calls[3]).toMatchObject({ path: "/api/v0/projects/B/issues", query: { priority: "low" } });
  await act(async () => {
    tab.nerve.calls[3]!.answer(json(200, issuePageOf([])));
  });
});
