import { IssueListService } from "@/services/issue/issue-list.service";
// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { RootStore } from "@/store/root.store";
import { GlobalViewStore } from "@/store/global-view.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { EViewAccess, type IWorkspaceView } from "@nerve/types";
import Layout from "./layout";
import GlobalViewIssuesPage from "./[globalViewId]/page";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useNavigate: () => vi.fn(),
  Outlet: () => <GlobalViewIssuesPage />,
}));
vi.mock("@/components/core/app-header", () => ({ AppHeader: ({ header }: { header: React.ReactNode }) => header }));
vi.mock("@/components/core/content-wrapper", () => ({
  ContentWrapper: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("@/components/workspace/views/modal", () => ({ CreateUpdateWorkspaceViewModal: () => null }));
vi.mock("@/components/workspace/views/quick-action", () => ({ WorkspaceViewQuickActions: () => null }));
vi.mock("@/components/workspace/views/default-view-quick-action", () => ({
  DefaultWorkspaceViewQuickActions: () => null,
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const container = document.createElement("div");
document.body.append(container);
const root = createRoot(container);
afterEach(async () => {
  await act(async () => root.render(null));
  vi.restoreAllMocks();
});

function createStore(nerve: FakeNerve) {
  return new RootStore(nerve.client());
}

it.each(["all-issues", "custom"])("preserves_domain_loading_without_issue_lists: workspace view %s", async (viewId) => {
  const nerve = new FakeNerve();
  const store = createStore(nerve);
  vi.useFakeTimers();
  await loadWorkspaces(nerve, store.workspaceRoot, [workspaceOf("ws")]);
  vi.useRealTimers();
  const custom: IWorkspaceView = {
    id: "custom",
    name: "Custom view",
    workspace: "id-ws",
    access: EViewAccess.PUBLIC,
    created_at: new Date(),
    updated_at: new Date(),
    is_favorite: false,
    created_by: "u",
    updated_by: "u",
    description: "A saved view",
    rich_filters: {},
    display_filters: {},
    display_properties: {},
    query: {},
    query_data: { rich_filters: {}, display_filters: {}, display_properties: {} },
    project: "p",
    is_locked: false,
    owned_by: "u",
  };
  runInAction(() => {
    store.router.setQuery({ workspaceSlug: "ws" });
    store.globalView.globalViewMap.custom = custom;
  });
  const fetch = vi
    .spyOn((store.globalView as GlobalViewStore).workspaceService, "getAllViews")
    .mockReturnValue(new Promise<never>(() => {}));
  const issues = vi.spyOn(IssueListService.prototype, "listProject");
  const swrConfig = { provider: () => new Map(), shouldRetryOnError: false };
  await act(async () =>
    root.render(
      <StoreContext.Provider value={store}>
        <SWRConfig value={swrConfig}>
          <Layout {...({ params: { workspaceSlug: "ws", globalViewId: viewId } } as ComponentProps<typeof Layout>)} />
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
  expect(fetch).toHaveBeenCalledOnce();
  expect(fetch).toHaveBeenCalledWith("ws");
  expect(document.title).toBe("ws - All Views");
  expect(issues).not.toHaveBeenCalled();
  const title = viewId === "custom" ? "Custom view" : "default_global_view.all_issues";
  const switcher = [...container.querySelectorAll("button")].find((button) => button.textContent?.includes(title));
  expect(switcher).toBeDefined();
  await act(async () => switcher?.click());
  expect(document.body.textContent).toContain("Custom view");
  expect(document.body.textContent).toContain("default_global_view.all_issues");
  store.dispose();
});

it("retains the missing-view state when the directory request fails", async () => {
  const nerve = new FakeNerve();
  const store = createStore(nerve);
  vi.spyOn((store.globalView as GlobalViewStore).workspaceService, "getAllViews").mockRejectedValue(
    new Error("not available")
  );
  const params = { workspaceSlug: "ws", globalViewId: "missing" };
  const swrConfig = { provider: () => new Map(), shouldRetryOnError: false };
  await act(async () => {
    root.render(
      <StoreContext.Provider value={store}>
        <SWRConfig value={swrConfig}>
          <Layout {...({ params } as ComponentProps<typeof Layout>)} />
        </SWRConfig>
      </StoreContext.Provider>
    );
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  expect(container.textContent).toContain("View does not exist");
  store.dispose();
});

it("shows the missing-view state when the current custom view is removed", async () => {
  const store = createStore(new FakeNerve());
  const view: IWorkspaceView = {
    id: "deleted",
    name: "Saved view",
    workspace: "id-ws",
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
    query_data: { rich_filters: {}, display_filters: {}, display_properties: {} },
    project: "p",
    is_locked: false,
    owned_by: "u",
  };
  runInAction(() => {
    store.globalView.globalViewMap.deleted = view;
  });
  vi.spyOn((store.globalView as GlobalViewStore).workspaceService, "getAllViews").mockRejectedValue(
    new Error("unavailable")
  );
  const swrConfig = { provider: () => new Map(), shouldRetryOnError: false };
  const params = { workspaceSlug: "ws", globalViewId: "deleted" };
  await act(async () => {
    root.render(
      <StoreContext.Provider value={store}>
        <SWRConfig value={swrConfig}>
          <Layout {...({ params } as ComponentProps<typeof Layout>)} />
        </SWRConfig>
      </StoreContext.Provider>
    );
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  expect(container.textContent).toContain("Saved view");
  await act(async () => {
    runInAction(() => {
      delete store.globalView.globalViewMap.deleted;
    });
  });
  expect(container.textContent).toContain("View does not exist");
  expect(container.textContent).toContain("Go to All work items");
  store.dispose();
});
