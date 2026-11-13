import { IssueListService } from "@/services/issue/issue-list.service";
// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { EViewAccess, type IProjectView } from "@nerve/types";
import { tokenManager } from "@/lib/auth/api-client";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import { RootStore } from "@/store/root.store";
import { RouterStore } from "@/store/router.store";
import { InstanceStore } from "@/store/instance.store";
import { ThemeStore } from "@/store/theme.store";
import { ProjectViewStore } from "@/store/project-view.store";
import { ModulePreferencesService } from "@/services/planning/module-preferences.service";
import { ModulesService } from "@/services/planning/modules.service";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { ModuleActionsHost } from "@/components/modules/actions-host";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProjects, moduleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import ModulePage from "./modules/(detail)/[moduleId]/page";
import { ModuleIssuesHeader } from "./modules/(detail)/header";
import { ProjectViewIssuesHeader } from "./views/(detail)/[viewId]/header";
import ViewPage from "./views/(detail)/[viewId]/page";

// the route's params, a module's page's naming its module
const route = vi.hoisted(() => ({ params: { workspaceSlug: "ws", projectId: "p-proj" } as Record<string, string> }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useNavigate: () => vi.fn(),
  useParams: () => route.params,
}));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => null }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
const container = document.createElement("div");
document.body.append(container);
const root = createRoot(container);
afterEach(async () => {
  await act(async () => root.render(null));
  vi.restoreAllMocks();
  route.params = { workspaceSlug: "ws", projectId: "p-proj" };
});

async function fixture() {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "ws", projectId: "p-proj" });
  const store = new RootStore(nerve.client(), { router, instance: new InstanceStore(), theme: new ThemeStore() });
  const workspace = workspaceOf("ws");
  vi.useFakeTimers();
  await loadWorkspaces(nerve, store.workspaceRoot, [workspace]);
  await loadProjects(nerve, store.projectRoot.project, workspace, [projectOf("PROJ", workspace.id)]);
  vi.useRealTimers();
  return store;
}

it("module details and its own loader, then the module's work items under the page's own preferences (P6b)", async () => {
  const params = { workspaceSlug: "ws", projectId: "p-proj", moduleId: "m" };
  route.params = params;
  const store = await fixture();
  const project = store.projectRoot.project.getProjectById("p-proj")!;
  const module = moduleOf(project, "Milestone", { id: "m", description: "Module details retained" });
  // the project's list holds the module (the project's wrapper read it); the page's own reads stay out
  vi.spyOn(ModulesService.prototype, "list").mockResolvedValue([moduleRowOf(module)]);
  await store.planning.modules.fetchModules("p-proj");
  const fetch = vi.spyOn(ModulesService.prototype, "get").mockReturnValue(new Promise<never>(() => {}));
  vi.spyOn(ModulesService.prototype, "progress").mockReturnValue(new Promise<never>(() => {}));
  vi.spyOn(ModulesService.prototype, "links").mockReturnValue(new Promise<never>(() => {}));
  vi.spyOn(tokenManager, "state", "get").mockReturnValue({ status: "signed-in", loginId: "x" });
  const { display_filters, display_properties } = preferencesOf();
  const preferences = vi
    .spyOn(ModulePreferencesService.prototype, "get")
    .mockResolvedValue({ filters: {}, display_filters, display_properties });
  const issues = vi.spyOn(IssueListService.prototype, "listProject").mockReturnValue(new Promise<never>(() => {}));
  await act(async () =>
    root.render(
      <MemoryRouter>
        <StoreContext.Provider value={store}>
          <SWRConfig value={{ provider: () => new Map() }}>
            <ModuleActionsHost workspaceSlug="ws" projectId="p-proj">
              <ModuleIssuesHeader {...params} />
              <ModulePage {...({ params } as ComponentProps<typeof ModulePage>)} />
            </ModuleActionsHost>
          </SWRConfig>
        </StoreContext.Provider>
      </MemoryRouter>
    )
  );
  expect(fetch).toHaveBeenCalledExactlyOnceWith("m");
  expect(preferences).toHaveBeenCalledExactlyOnceWith("m");
  await vi.waitFor(() =>
    expect(issues).toHaveBeenCalledExactlyOnceWith("p-proj", expect.objectContaining({ module_id: ["m"] }))
  );
  expect(document.title).toBe("PROJ - Milestone");
  expect(container.textContent).toContain("Milestone");
  expect(container.textContent).not.toContain("Add work item");
  expect(container.textContent).not.toContain("Display");
  expect(container.textContent).toContain("Module details retained");
  store.dispose();
});

it("preserves_domain_loading_without_issue_lists: project view has one detail reader", async () => {
  const store = await fixture();
  const view: IProjectView = {
    id: "v",
    name: "Release view",
    access: EViewAccess.PUBLIC,
    created_at: new Date(),
    updated_at: new Date(),
    is_favorite: false,
    created_by: "u",
    updated_by: "u",
    description: "",
    rich_filters: {},
    display_filters: {},
    display_properties: {},
    query: {},
    query_data: {},
    project: "p-proj",
    workspace: "id-ws",
    logo_props: undefined,
    is_locked: false,
    owned_by: "u",
  };
  runInAction(() => {
    store.projectView.viewMap.v = view;
  });
  const fetch = vi
    .spyOn((store.projectView as ProjectViewStore).viewService, "getViewDetails")
    .mockReturnValue(new Promise<never>(() => {}));
  const issues = vi.spyOn(IssueListService.prototype, "listProject");
  const params = { workspaceSlug: "ws", projectId: "p-proj", viewId: "v" };
  await act(async () =>
    root.render(
      <MemoryRouter>
        <StoreContext.Provider value={store}>
          <SWRConfig value={{ provider: () => new Map() }}>
            <ProjectViewIssuesHeader {...params} />
            <ViewPage {...({ params } as ComponentProps<typeof ViewPage>)} />
          </SWRConfig>
        </StoreContext.Provider>
      </MemoryRouter>
    )
  );
  expect(fetch).toHaveBeenCalledExactlyOnceWith("ws", "p-proj", "v");
  expect(issues).not.toHaveBeenCalled();
  expect(document.title).toBe("PROJ - Release view");
  expect(container.textContent).toContain("Release view");
  expect(container.textContent).not.toContain("Add work item");
  expect(container.textContent).not.toContain("Display");
  store.dispose();
});
