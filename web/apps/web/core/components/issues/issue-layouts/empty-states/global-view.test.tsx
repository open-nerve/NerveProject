// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's view's page with no work items to show (M7/P6b spec 2.3 point 9, 2.7, 2.8, R24), by its store type,
// GLOBAL: with conditions applied, the search's empty state, whose clearing ends them; with none, when the caller is
// an active member of no unarchived project of the workspace, a new project; else a new work item of the page. The two
// creations are offered to the workspace's admins and members, never its guests (nerve's project.create and work item
// rules). The stores, the layouts, SWR and HTTP are real (fake-view-page.tsx).
import { act } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { until } from "@/lib/auth/fake-time";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import { loadProjects } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { WorkspaceViewLayoutRoot } from "../roots/workspace-view-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", globalViewId: "all-issues" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));

const ALL = workspacePageId("ws", "all-issues");
let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
beforeEach(() => {
  window.localStorage.clear();
});
afterEach(async () => page.close());

/** All work items' page with the caller of role, none listed: as fields say, with a condition, or with no projects. */
async function shown(role: WorkspaceRole, fields: { condition?: boolean; noProjects?: boolean } = {}) {
  page = await workspaceViewPageTab(role);
  bound.app = page.app;
  // the caller sees A and B, a member of neither
  if (fields.noProjects)
    await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [
      { ...page.project, member_role: null },
      { ...page.project, id: "B", identifier: "B", member_role: null, archived_at: null },
    ]);
  const store = page.app.issue.globalIssuesFilter;
  await store.fetchFilters("ws", ALL);
  if (fields.condition) await store.updateFilterExpression("ws", ALL, { and: [{ priority__in: "high" }] });
  page.nerve.calls.length = 0;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <WorkspaceViewLayoutRoot workspaceSlug="ws" globalViewId="all-issues" />
      </MemoryRouter>
    </SWRConfig>
  );
  await page.answerList([]);
}
const text = () => document.body.textContent ?? "";
const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === name);
const palette = () => page.app.commandPalette;

it.each([
  ["its admin", 20],
  ["its guest", 5],
] as const)("tells %s the conditions match nothing, and clears them, listing again without them", async (_, role) => {
  await shown(role, { condition: true });
  expect(text()).toContain("common_empty_state.search.title");
  const clear = button("project_issues.empty_state.issues_empty_filter.secondary_button.text");
  expect(clear?.disabled).toBe(false);
  await act(async () => clear?.click());
  await until(() => page.lists().length === 2, "the list again");
  expect(page.lists()[1]!.query.priority).toBeUndefined();
  expect(page.writes()).toEqual([]);
});

it.each([
  ["its admin", 20, true],
  ["its member", 15, true],
  ["its guest", 5, false],
] as const)("asks %s for a project when he is in none of the workspace's: offered %s", async (_, role, offered) => {
  await shown(role, { noProjects: true });
  expect(text()).toContain("workspace_projects.empty_state.no_projects.title");
  const create = button("workspace_projects.empty_state.no_projects.primary_button.text");
  expect(create !== undefined).toBe(offered);
  await act(async () => create?.click());
  expect(palette().isCreateProjectModalOpen).toBe(offered);
});

it.each([
  ["its admin", 20, true],
  ["its member", 15, true],
  ["its guest", 5, false],
] as const)("offers %s a new work item of the page when nothing is listed: %s", async (_, role, offered) => {
  await shown(role);
  expect(text()).toContain("workspace_empty_state.views.title");
  const create = button("workspace_empty_state.views.cta_primary");
  expect(create !== undefined).toBe(offered);
  await act(async () => create?.click());
  expect([palette().isCreateIssueModalOpen, palette().createIssueStoreType]).toEqual(
    offered ? [true, EIssuesStoreType.GLOBAL] : [false, EIssuesStoreType.PROJECT]
  );
});
