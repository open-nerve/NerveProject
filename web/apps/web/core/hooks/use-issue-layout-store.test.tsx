// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { json } from "@/lib/auth/fake-nerve";
import { IssuesStoreContext, useIssueStoreType } from "./use-issue-layout-store";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const fixture = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  app: undefined as RootStore | undefined,
}));
vi.mock("react-router", () => ({ useParams: () => fixture.params }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "actions" } }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
  get rootStore() {
    return fixture.app;
  },
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.params = {};
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function mount(run: () => void, context?: EIssuesStoreType, use?: () => Promise<void>) {
  const root = createRoot(document.createElement("div"));
  const Probe = () => {
    run();
    return null;
  };
  try {
    await act(async () =>
      root.render(
        <StoreContext.Provider value={tab.app}>
          <IssuesStoreContext.Provider value={context}>
            <Probe />
          </IssuesStoreContext.Provider>
        </StoreContext.Provider>
      )
    );
    await use?.();
  } finally {
    await act(async () => root.unmount());
  }
}

it.each(["globalViewId", "viewId", "cycleId", "moduleId"])(
  "uses generic commands on %s routes and creates in the selected project without fetching a list",
  async (route) => {
    const selected = projectOf("SELECTED", tab.workspace.id, { id: "selected-project" });
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, selected]);
    tab.nerve.calls.length = 0;
    fixture.params = {
      workspaceSlug: "ws",
      ...(route === "globalViewId" ? {} : { projectId: "A" }),
      [route]: "domain-id",
    };
    let actions!: IssueActions;
    await mount(
      () => {
        const selectedType = useIssueStoreType();
        expect(selectedType).toBe(EIssuesStoreType.DEFAULT);
        actions = useIssuesActions(selectedType);
      },
      undefined,
      async () => {
        const response = { ...issueDetailOf("new", selected.id), name: "From global create" };
        const sent = await requestedIssue(tab.nerve, () => actions.createIssue!(selected.id, { name: response.name }), [
          "POST",
          "/api/v0/projects/selected-project/issues",
        ]);
        expect(sent.call.body).toEqual({ name: response.name });
        sent.call.answer(json(201, response));
        await expect(sent.promise).resolves.toEqual(response);
      }
    );
    expect(tab.nerve.calls.map(({ method, path }) => [method, path])).toEqual([
      ["POST", "/api/v0/projects/selected-project/issues"],
    ]);
  }
);
it("keeps project/profile route selection, explicit context and a generic route without project", async () => {
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.DEFAULT));
  fixture.params = { projectId: "A" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.PROJECT));
  fixture.params = { userId: "user" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.PROFILE));
  fixture.params = { globalViewId: "view" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.ARCHIVED), EIssuesStoreType.ARCHIVED);
});
it.each(["GLOBAL", "CYCLE", "MODULE", "PROJECT_VIEW", "unknown"])(
  "rejects %s actions instead of silently dispatching project commands",
  async (kind) => {
    // Deliberately invalid runtime input verifies the dispatch guard.
    await mount(() => expect(() => useIssuesActions(kind as EIssuesStoreType)).toThrow("Unsupported issue store"));
    expect(tab.nerve.calls).toEqual([]);
  }
);
it("does not fetch lists for unknown scopes or nonmembers", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, []);
  tab.nerve.calls.length = 0;
  fixture.params = { workspaceSlug: "ws", projectId: "A", userId: "actor" };
  await mount(() => {
    void useIssuesActions(EIssuesStoreType.PROJECT).fetchIssues("init-loader", { limit: 20 });
    void useIssuesActions(EIssuesStoreType.ARCHIVED).fetchIssues("init-loader", { limit: 20 });
    void useIssuesActions(EIssuesStoreType.PROFILE).fetchIssues("init-loader", { limit: 20 }, "assigned");
  });
  expect(tab.nerve.calls).toEqual([]);
});
it("does not fetch archive or profile lists for guests", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  tab.nerve.calls.length = 0;
  fixture.params = { workspaceSlug: "ws", projectId: "A", userId: "actor" };
  await mount(() => {
    void useIssuesActions(EIssuesStoreType.ARCHIVED).fetchIssues("init-loader", { limit: 20 });
    void useIssuesActions(EIssuesStoreType.PROFILE).fetchIssues("init-loader", { limit: 20 }, "assigned");
  });
  expect(tab.nerve.calls).toEqual([]);
});
