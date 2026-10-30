// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { GroupByColumnTypes } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects, projectOf, labelOf, stateOf, projectMemberOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { until } from "@/lib/auth/fake-time";
import { ListGroup } from "./list/list-group";
import { KanBan } from "./kanban/default";
import { answered } from "@/lib/auth/fake-nerve";
import { getGroupByColumns } from "./group-columns";
import { IssuesHeader } from "../header";
import { ProjectEmptyState } from "./empty-states/project-issues";
import { ProjectArchivedEmptyState } from "./empty-states/archived-issues";
const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  route: { workspaceSlug: "ws", projectId: "B" as string | undefined },
}));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.route,
}));
// No viewport geometry in jsdom; these cases exercise directory-backed prefill.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// Full filter editing and project selection have their own tests; retain the actual list breadcrumb and actions.
vi.mock("@/components/issues/filters", () => ({ HeaderFilters: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let b: Project;
let container: HTMLDivElement;
let renderer: Root;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  bound.app = tab.app;
  bound.route = { workspaceSlug: "ws", projectId: "B" };
  const other = workspaceOf("other");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [
    { ...tab.workspace, role: 15 },
    { ...other, role: 15 },
  ]);
  b = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, b]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [projectOf("C", other.id, { id: "C" })]);
  const loadDirectories = async (project: Project) => {
    await answered(
      tab.nerve,
      () => tab.app.state.fetchProjectStates(project.id),
      ["GET", `/api/v0/projects/${project.id}/states`],
      { data: [stateOf(project, `${project.id} state`, "unstarted", 1, { default: true })] },
      "states"
    );
    await answered(
      tab.nerve,
      () => tab.app.label.fetchProjectLabels(project.id),
      ["GET", `/api/v0/projects/${project.id}/labels`],
      { data: [labelOf(project, `${project.id} label`, 1)] },
      "labels"
    );
    await answered(
      tab.nerve,
      () => tab.app.memberRoot.project.fetchProjectMembers(project.id),
      ["GET", `/api/v0/projects/${project.id}/members`],
      { data: [projectMemberOf(project, project.id)] },
      "members"
    );
  };
  await loadDirectories(tab.project);
  await loadDirectories(b);
  await answered(
    tab.nerve,
    () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(tab.workspace),
    ["GET", "/api/v0/workspaces/ws/members"],
    { data: [membershipOf("A", {}, "ws"), membershipOf("B", {}, "ws")] },
    "workspace members"
  );
  await answered(
    tab.nerve,
    () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(other),
    ["GET", "/api/v0/workspaces/other/members"],
    { data: [membershipOf("C", {}, "other")] },
    "other members"
  );
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
  tab.nerve.calls.length = 0;
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const columns = (
  groupBy: GroupByColumnTypes,
  projectId: string | undefined = "B",
  workspaceSlug = "ws",
  workspace = false
) => getGroupByColumns({ groupBy, includeNone: true, isWorkspaceLevel: workspace, workspaceSlug, projectId });
const render = () =>
  act(async () =>
    renderer.render(
      <MemoryRouter>
        <StoreContext.Provider value={tab.app}>
          <IssuesHeader />
          <ProjectEmptyState />
          <ProjectArchivedEmptyState />
        </StoreContext.Provider>
      </MemoryRouter>
    )
  );
it("rendersTheNewRouteBeforeRouterStoreEffect: every project directory comes from B while the store stays A", () => {
  expect(tab.app.router.projectId).toBe("A");
  expect(columns("state")?.map((column) => column.name)).toEqual(["B state"]);
  expect(columns("labels")?.map((column) => column.name)).toEqual(["B label", "None"]);
  expect(columns("assignees")?.map((column) => column.id)).toEqual(["u-B", "none"]);
  expect(columns("created_by")?.map((column) => column.id)).toEqual(["u-B"]);
  expect(tab.nerve.calls).toHaveLength(0);
});
it.each([15, 5] as const)(
  "rendersTheNewRouteBeforeRouterStoreEffect: route B role %s controls real header and empty actions",
  async (role) => {
    tab.app.projectRoot.project.confirmProject(b, (project) => ({ ...project, member_role: role }));
    await render();
    expect(container.querySelector('a[href="/ws/projects/B/issues"]')).not.toBeNull();
    const create = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "project_empty_state.work_items.cta_primary"
    )!;
    expect(create.disabled).toBe(role === 5);
    const headerCreate = [...container.querySelectorAll("button")].find((button) =>
      button.textContent?.includes("issue.add.label")
    );
    expect(!!headerCreate).toBe(role === 15);
    expect(tab.app.router.projectId).toBe("A");
    expect(tab.nerve.calls).toHaveLength(0);
  }
);
it("missing project scope never falls back to the router's A", async () => {
  bound.route.projectId = undefined;
  expect(
    getGroupByColumns({
      groupBy: "state",
      includeNone: true,
      isWorkspaceLevel: false,
      workspaceSlug: "ws",
      projectId: undefined,
    })
  ).toBeUndefined();
  await render();
  const create = [...container.querySelectorAll("button")].find(
    (button) => button.textContent === "project_empty_state.work_items.cta_primary"
  )!;
  expect(create.disabled).toBe(true);
  expect(tab.nerve.calls).toHaveLength(0);
});
it("an explicit workspace never borrows the router's projects or members and helpers remain synchronous", async () => {
  expect(columns("project", undefined, "other", true)?.map((column) => column.id)).toEqual(["C"]);
  expect(columns("assignees", undefined, "other", true)?.map((column) => column.id)).toEqual(["u-C", "none"]);
  expect(columns("created_by", undefined, "other", true)?.map((column) => column.id)).toEqual(["u-C"]);
  expect(columns("labels", "B", "other")).toBeUndefined();
  await act(async () => {});
  expect(tab.nerve.calls).toHaveLength(0);
});

it.each(["list", "kanban"] as const)(
  "%s default quick-add state follows route B before router store's effect",
  async (layout) => {
    const common = {
      orderBy: "sort_order" as const,
      updateIssue: undefined,
      quickActions: () => null,
      displayProperties: undefined,
      canEditProperties: () => true,
      handleOnDrop: async () => {},
      collapsedGroups: { group_by: [], sub_group_by: [] },
      handleCollapsedGroups: () => {},
      loadMoreIssues: () => {},
      showEmptyGroup: true,
      quickAddCallback: (project: string | null | undefined, body: import("@nerve/api-client").IssueCreate) =>
        tab.app.issue.createIssue(tab.app.issue.scope("ws", project!), body),
    };
    await act(async () =>
      renderer.render(
        <MemoryRouter>
          <StoreContext.Provider value={tab.app}>
            {layout === "list" ? (
              <ListGroup
                {...common}
                group={{ id: "", name: "All", payload: {} }}
                group_by={null}
                groupIssueIds={[]}
                issuesMap={{}}
                getGroupIndex={() => 0}
                enableIssueQuickAdd
                containerRef={{ current: null }}
              />
            ) : (
              <KanBan
                {...common}
                groups={new Map()}
                issuesMap={{}}
                getGroupIssueCount={() => 0}
                group_by="priority"
                sub_group_by={undefined}
                enableQuickIssueCreate
              />
            )}
          </StoreContext.Provider>
        </MemoryRouter>
      )
    );
    const open = [...container.querySelectorAll("span")].find((node) => node.textContent === "issue.new")!;
    expect(open.closest("button"), "quick-add has a named keyboard trigger").not.toBeNull();
    await act(async () => open.click());
    const input = container.querySelector("input")!;
    expect(document.activeElement).toBe(input);
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New in B");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () =>
      container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
    await until(() => tab.nerve.calls.length === 1, "B create");
    expect(tab.nerve.calls[0]).toMatchObject({
      method: "POST",
      path: "/api/v0/projects/B/issues",
      body: { state_id: tab.app.state.getProjectStates("B")![0]!.id },
    });
    expect(tab.app.router.projectId).toBe("A");
  }
);
