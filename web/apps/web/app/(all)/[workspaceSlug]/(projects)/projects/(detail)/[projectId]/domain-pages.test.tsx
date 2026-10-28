import { IssueListService } from "@/services/issue/issue-list.service";
// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { EViewAccess, type IModule, type IProjectView } from "@nerve/types";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import { RootStore } from "@/store/root.store";
import { RouterStore } from "@/store/router.store";
import { InstanceStore } from "@/store/instance.store";
import { ThemeStore } from "@/store/theme.store";
import { ModulesStore } from "@/store/module.store";
import { ProjectViewStore } from "@/store/project-view.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import ModulePage from "./modules/(detail)/[moduleId]/page";
import { ModuleIssuesHeader } from "./modules/(detail)/header";
import { ProjectViewIssuesHeader } from "./views/(detail)/[viewId]/header";
import ViewPage from "./views/(detail)/[viewId]/page";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useNavigate: () => vi.fn(),
  useParams: () => ({ workspaceSlug: "ws", projectId: "p-proj" }),
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

it("preserves_domain_loading_without_issue_lists: module details and its own loader", async () => {
  const store = await fixture();
  const module: IModule = {
    id: "m",
    name: "Milestone",
    description: "Module details retained",
    description_text: "",
    description_html: "",
    workspace_id: "id-ws",
    project_id: "p-proj",
    lead_id: null,
    member_ids: [],
    is_favorite: false,
    sort_order: 0,
    view_props: { filters: {} },
    archived_at: null,
    start_date: null,
    target_date: null,
    created_at: "",
    updated_at: "",
    total_issues: 0,
    completed_issues: 0,
    backlog_issues: 0,
    started_issues: 0,
    unstarted_issues: 0,
    cancelled_issues: 0,
  };
  runInAction(() => {
    store.module.moduleMap.m = module;
  });
  const fetch = vi
    .spyOn((store.module as ModulesStore).moduleService, "getModuleDetails")
    .mockReturnValue(new Promise<never>(() => {}));
  const issues = vi.spyOn(IssueListService.prototype, "listProject");
  const params = { workspaceSlug: "ws", projectId: "p-proj", moduleId: "m" };
  await act(async () =>
    root.render(
      <MemoryRouter>
        <StoreContext.Provider value={store}>
          <SWRConfig value={{ provider: () => new Map() }}>
            <ModuleIssuesHeader {...params} />
            <ModulePage {...({ params } as ComponentProps<typeof ModulePage>)} />
          </SWRConfig>
        </StoreContext.Provider>
      </MemoryRouter>
    )
  );
  expect(fetch).toHaveBeenCalledExactlyOnceWith("ws", "p-proj", "m");
  expect(issues).not.toHaveBeenCalled();
  expect(document.title).toBe("PROJ - Milestone");
  expect(container.textContent).toContain("Milestone");
  expect(container.textContent).not.toContain("Add work item");
  expect(container.textContent).not.toContain("Display");
  expect(container.querySelector("textarea")?.value).toBe("Module details retained");
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
