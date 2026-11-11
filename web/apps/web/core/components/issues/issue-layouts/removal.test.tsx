import { IssueListService } from "@/services/issue/issue-list.service";
// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { runInAction } from "mobx";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import type { IModule } from "@nerve/types";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import { RootStore } from "@/store/root.store";
import { rowOf } from "@/store/planning/fake-cycles";
import { ModulesStore } from "@/store/module.store";
import { RouterStore } from "@/store/router.store";
import { InstanceStore } from "@/store/instance.store";
import { ThemeStore } from "@/store/theme.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { cycleOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import { CycleActionsHost } from "@/components/cycles/actions-host";
import { ArchivedCycleLayoutRoot } from "@/components/cycles/archived-cycles/root";
import { ArchivedModuleLayoutRoot } from "@/components/modules/archived-modules/root";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "p-proj" }),
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
const container = document.createElement("div");
document.body.append(container);
const root = createRoot(container);
let store: RootStore | undefined;
afterEach(async () => {
  await act(async () => root.render(null));
  store?.dispose();
  vi.restoreAllMocks();
});

async function fixture(nerve = new FakeNerve()) {
  const router = new RouterStore();
  store = new RootStore(nerve.client(), { router, instance: new InstanceStore(), theme: new ThemeStore() });
  // The real app changes the route after constructing the stores; the domain-filter reactions initialize it.
  router.setQuery({ workspaceSlug: "ws", projectId: "p-proj" });
  const workspace = workspaceOf("ws");
  vi.useFakeTimers();
  await loadWorkspaces(nerve, store.workspaceRoot, [workspace]);
  await loadProjects(nerve, store.projectRoot.project, workspace, [projectOf("PROJ", workspace.id)]);
  vi.useRealTimers();
  return store;
}

const cycle = rowOf(
  cycleOf(projectOf("PROJ", "id-ws"), "Archived spring", {
    id: "archived-cycle",
    description: "Retained cycle",
    owned_by_id: "u",
    sort_order: 1,
    status: "completed",
    archived_at: "2026-10-01T09:00:00Z",
  }),
  { total_issues: 2, completed_issues: 1, backlog_issues: 1 }
);
const module: IModule = {
  id: "archived-module",
  name: "Archived milestone",
  description: "Retained module",
  description_text: "",
  description_html: "",
  workspace_id: "id-ws",
  project_id: "p-proj",
  lead_id: null,
  member_ids: [],
  is_favorite: false,
  sort_order: 0,
  view_props: { filters: {} },
  archived_at: "2026-10-01T09:00:00Z",
  start_date: null,
  target_date: null,
  created_at: "2026-09-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
  total_issues: 2,
  completed_issues: 1,
  backlog_issues: 1,
  started_issues: 0,
  unstarted_issues: 0,
  cancelled_issues: 0,
};

it("preserves_domain_loading_without_issue_lists: archived cycle retains real directory items and search", async () => {
  const nerve = new FakeNerve();
  // nerve lists the project's archived cycles (P5a spec 2.5): the archive reads them itself
  nerve.respond = (call) => {
    if (call.path === "/api/v0/projects/p-proj/cycles" && call.query.archived === "true")
      call.answer(json(200, { data: [cycle] }));
  };
  const current = await fixture(nerve);
  const issues = vi.spyOn(IssueListService.prototype, "listProject");
  await act(async () => {
    root.render(
      <MemoryRouter>
        <StoreContext.Provider value={current}>
          <SWRConfig value={{ provider: () => new Map() }}>
            <CycleActionsHost workspaceSlug="ws" projectId="p-proj">
              <ArchivedCycleLayoutRoot workspaceSlug="ws" projectId="p-proj" />
            </CycleActionsHost>
          </SWRConfig>
        </StoreContext.Provider>
      </MemoryRouter>
    );
  });
  await act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  const archive = () => nerve.calls.filter((call) => call.path === "/api/v0/projects/p-proj/cycles");
  expect(archive().map((call) => call.query)).toEqual([{ archived: "true" }]);
  expect(container.textContent).toContain(cycle.name);
  expect(container.querySelector(`a[href="/ws/projects/p-proj/cycles/${cycle.id}"]`)).not.toBeNull();
  expect(container.textContent).not.toContain("Add work item");
  expect(issues).not.toHaveBeenCalled();
  await act(async () => current.planning.cycleFilters.updateSearchQuery("p-proj", "archived", "unmatched"));
  expect(container.textContent).not.toContain(cycle.name);
  expect(container.textContent).toContain("project_cycles.no_matching_cycles");
  expect(archive()).toHaveLength(1);
  expect(issues).not.toHaveBeenCalled();
});

it("preserves_domain_loading_without_issue_lists: archived module retains real directory items and search", async () => {
  const current = await fixture();
  runInAction(() => {
    current.module.moduleMap[module.id] = module;
    (current.module as ModulesStore).fetchedMap["p-proj"] = true;
  });
  const fetch = vi.spyOn((current.module as ModulesStore).moduleArchiveService, "getArchivedModules");
  // Valid entities are store fixtures, not fabricated HTTP success. Keep the real store's failure path too.
  fetch.mockRejectedValue(new Error("M6 archive API unavailable"));
  const issues = vi.spyOn(IssueListService.prototype, "listProject");
  await act(async () => {
    root.render(
      <MemoryRouter>
        <StoreContext.Provider value={current}>
          <SWRConfig value={{ provider: () => new Map() }}>
            <ArchivedModuleLayoutRoot />
          </SWRConfig>
        </StoreContext.Provider>
      </MemoryRouter>
    );
  });
  const name = module.name;
  expect(fetch).toHaveBeenCalledExactlyOnceWith("ws", "p-proj");
  expect(container.textContent).toContain(name);
  expect(container.querySelector(`a[href="/ws/projects/p-proj/modules/${module.id}"]`)).not.toBeNull();
  expect(container.textContent).not.toContain("Add work item");
  expect(issues).not.toHaveBeenCalled();
  await act(async () => {
    current.moduleFilter.updateArchivedModulesSearchQuery("unmatched");
  });
  expect(container.textContent).not.toContain(name);
  expect(container.textContent).toContain("No matching modules");
  expect(fetch).toHaveBeenCalledOnce();
  expect(issues).not.toHaveBeenCalled();
});
