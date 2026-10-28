import { IssueListService } from "@/services/issue/issue-list.service";
// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { runInAction } from "mobx";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import type { ICycle, IModule } from "@nerve/types";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import { RootStore } from "@/store/root.store";
import { CycleStore } from "@/store/cycle.store";
import { ModulesStore } from "@/store/module.store";
import { RouterStore } from "@/store/router.store";
import { InstanceStore } from "@/store/instance.store";
import { ThemeStore } from "@/store/theme.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { ArchivedCycleLayoutRoot } from "@/components/cycles/archived-cycles/root";
import { ArchivedModuleLayoutRoot } from "@/components/modules/archived-modules/root";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
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

async function fixture() {
  const nerve = new FakeNerve();
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

const cycle: ICycle = {
  id: "archived-cycle",
  name: "Archived spring",
  description: "Retained cycle",
  project_id: "p-proj",
  workspace_id: "id-ws",
  owned_by_id: "u",
  sort_order: 1,
  start_date: null,
  end_date: null,
  archived_at: "2026-10-01T09:00:00Z",
  project_detail: { id: "p-proj" },
  view_props: { filters: {} },
  progress_snapshot: undefined,
  total_issues: 2,
  completed_issues: 1,
  backlog_issues: 1,
  started_issues: 0,
  unstarted_issues: 0,
  cancelled_issues: 0,
};
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

it.each(["cycle", "module"] as const)(
  "preserves_domain_loading_without_issue_lists: archived %s retains real directory items and search",
  async (domain) => {
    const current = await fixture();
    runInAction(() => {
      current.cycle.cycleMap[cycle.id] = cycle;
      (current.cycle as CycleStore).fetchedMap["p-proj"] = true;
      current.module.moduleMap[module.id] = module;
      (current.module as ModulesStore).fetchedMap["p-proj"] = true;
    });
    const fetch =
      domain === "cycle"
        ? vi.spyOn((current.cycle as CycleStore).cycleArchiveService, "getArchivedCycles")
        : vi.spyOn((current.module as ModulesStore).moduleArchiveService, "getArchivedModules");
    // Valid entities are store fixtures, not fabricated HTTP success. Keep the real store's failure path too.
    fetch.mockRejectedValue(new Error("M6 archive API unavailable"));
    const issues = vi.spyOn(IssueListService.prototype, "listProject");
    await act(async () => {
      root.render(
        <MemoryRouter>
          <StoreContext.Provider value={current}>
            <SWRConfig value={{ provider: () => new Map() }}>
              {domain === "cycle" ? <ArchivedCycleLayoutRoot /> : <ArchivedModuleLayoutRoot />}
            </SWRConfig>
          </StoreContext.Provider>
        </MemoryRouter>
      );
    });
    const name = domain === "cycle" ? cycle.name : module.name;
    expect(fetch).toHaveBeenCalledExactlyOnceWith("ws", "p-proj");
    expect(container.textContent).toContain(name);
    expect(
      container.querySelector(
        `a[href="/ws/projects/p-proj/${domain === "cycle" ? "cycles" : "modules"}/${domain === "cycle" ? cycle.id : module.id}"]`
      )
    ).not.toBeNull();
    expect(container.textContent).not.toContain("Add work item");
    expect(issues).not.toHaveBeenCalled();
    await act(async () => {
      if (domain === "cycle") current.cycleFilter.updateArchivedCyclesSearchQuery("unmatched");
      else current.moduleFilter.updateArchivedModulesSearchQuery("unmatched");
    });
    expect(container.textContent).not.toContain(name);
    expect(container.textContent).toContain(domain === "cycle" ? "No matching cycles" : "No matching modules");
    expect(fetch).toHaveBeenCalledOnce();
    expect(issues).not.toHaveBeenCalled();
  }
);
