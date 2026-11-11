// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { runInAction } from "mobx";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { EIssueFilterType } from "@nerve/constants";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { cycleOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { answered, json, noContent } from "@/lib/auth/fake-nerve";
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
async function mount(
  run: () => void,
  context?: EIssuesStoreType,
  use?: (render: () => Promise<void>) => Promise<void>
) {
  const root = createRoot(document.createElement("div"));
  const Probe = () => {
    run();
    return null;
  };
  const render = () =>
    act(async () =>
      root.render(
        <StoreContext.Provider value={tab.app}>
          <IssuesStoreContext.Provider value={context}>
            <Probe />
          </IssuesStoreContext.Provider>
        </StoreContext.Provider>
      )
    );
  try {
    await render();
    await use?.(render);
  } finally {
    await act(async () => root.unmount());
  }
}

it.each(["globalViewId", "viewId", "cycleId", "moduleId"])(
  "selects the %s route's commands and creates in the selected project without fetching a list",
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
        expect(selectedType).toBe(route === "cycleId" ? EIssuesStoreType.CYCLE : EIssuesStoreType.DEFAULT);
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
it.each(["GLOBAL", "MODULE", "PROJECT_VIEW", "unknown"])(
  "rejects %s actions instead of silently dispatching project commands",
  async (kind) => {
    // Deliberately invalid runtime input verifies the dispatch guard.
    await mount(() => expect(() => useIssuesActions(kind as EIssuesStoreType)).toThrow("Unsupported issue store"));
    expect(tab.nerve.calls).toEqual([]);
  }
);
const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
// P5b spec 2.5: the cycle page's commands act for the cycle in its route params; the router store is never read.
it("cycle actions list, create and keep preferences for the route's cycle, never the router store's", async () => {
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", cycleId: "router-cycle" }));
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  let actions!: IssueActions;
  await mount(
    () => {
      actions = useIssuesActions(EIssuesStoreType.CYCLE);
    },
    EIssuesStoreType.CYCLE,
    async (render) => {
      const list = await requestedIssue(tab.nerve, () => actions.fetchIssues("init-loader", { limit: 1 }), [
        "GET",
        "/api/v0/projects/A/issues",
      ]);
      expect(list.call.query.cycle_id).toBe(C1);
      list.call.answer(json(200, { ...issuePageOf([{ ...issueOf("a"), cycle_id: C1 }], "next"), total: 2 }));
      await list.promise;
      fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C2 };
      await render();
      // The first cycle's list is still the store's current one; the next page belongs to the cycle left.
      await expect(actions.fetchNextIssues()).resolves.toBeUndefined();
      expect(tab.nerve.calls).toHaveLength(1);
      // A creation joins the route's cycle while the page shows it taking work items (T9): nerve has given it.
      const next = cycleOf(tab.project, "Next", { id: C2, status: "current" });
      await answered(
        tab.nerve,
        () => tab.app.planning.cycles.fetchCycle(C2),
        ["GET", `/api/v0/cycles/${C2}`],
        next,
        "C2"
      );
      const created = await requestedIssue(tab.nerve, () => actions.createIssue!("A", { name: "New" }), [
        "POST",
        "/api/v0/projects/A/issues",
      ]);
      expect(created.call.body).toEqual({ name: "New", cycle_id: C2 });
      created.call.answer(json(201, { ...issueDetailOf("new"), cycle_id: C2 }));
      await created.promise;
      // The change is followed as the project page's (T8): the store's list being the cycle left's, not the page's,
      // the route cycle's list is read again with the pages it has.
      const kept = actions.updateFilters("A", EIssueFilterType.KANBAN_FILTERS, {
        group_by: ["done"],
        sub_group_by: [],
      });
      expect(tab.app.issue.cycleIssuesFilter.filters[C2]?.kanbanFilters?.group_by).toEqual(["done"]);
      expect(tab.app.issue.cycleIssuesFilter.filters["router-cycle"]).toBeUndefined();
      expect(tab.app.issue.projectIssuesFilter.filters.A).toBeUndefined();
      const followed = await requestedIssue(tab.nerve, () => kept, ["GET", "/api/v0/projects/A/issues"]);
      expect(followed.call.query.cycle_id).toBe(C2);
      followed.call.answer(json(200, issuePageOf([])));
      await followed.promise;
      await answered(
        tab.nerve,
        () => actions.fetchIssues("init-loader", { limit: 1 }),
        ["GET", "/api/v0/projects/A/issues"],
        issuePageOf([]),
        "second cycle"
      );
      expect(tab.nerve.calls.at(-1)?.query.cycle_id).toBe(C2);
    }
  );
});
it("reads a cycle's list for any member of its project, guests included, and never for a nonmember", async () => {
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: null }]);
  tab.nerve.calls.length = 0;
  await mount(() => {
    void useIssuesActions(EIssuesStoreType.CYCLE).fetchIssues("init-loader", { limit: 20 });
    void useIssuesActions(EIssuesStoreType.CYCLE).fetchNextIssues();
  });
  expect(tab.nerve.calls).toEqual([]);
  // Nothing is even selected: the binding would refuse the read, but the page never asks it.
  expect(tab.app.issue.cycleIssues.currentList).toBeUndefined();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  tab.nerve.calls.length = 0;
  let actions!: IssueActions;
  await mount(
    () => {
      actions = useIssuesActions(EIssuesStoreType.CYCLE);
    },
    EIssuesStoreType.CYCLE,
    () =>
      answered(
        tab.nerve,
        () => actions.fetchIssues("init-loader", { limit: 20 }),
        ["GET", "/api/v0/projects/A/issues"],
        issuePageOf([]),
        "guest's cycle list"
      ).then(() => undefined)
  );
  expect(tab.nerve.calls.map(({ query }) => query.cycle_id)).toEqual([C1]);
});
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

it.each(["restore", "delete", "collapse"] as const)(
  "archive %s follows the current workspace after the mounted hook changes routes",
  async (operation) => {
    const other = workspaceOf("other");
    const project = projectOf("B", other.id, { id: "B" });
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, other, [project]);
    const filters = tab.app.issue.archivedIssuesFilter;
    await filters.fetchFilters("ws", "A");
    await filters.fetchFilters("other", "B");
    const oldFilters = JSON.stringify(filters.getIssueFilters("A", "ws"));
    const archived = {
      ...issueDetailOf("old", "B"),
      workspace_id: other.id,
      archived_at: "2026-10-30T00:00:00Z",
    };
    tab.nerve.calls.length = 0;
    fixture.params = { workspaceSlug: "ws", projectId: "A" };
    let actions!: IssueActions;
    await mount(
      () => {
        actions = useIssuesActions(EIssuesStoreType.ARCHIVED);
      },
      EIssuesStoreType.ARCHIVED,
      async (render) => {
        fixture.params = { workspaceSlug: "other", projectId: "B" };
        await render();
        if (operation === "collapse") {
          await actions.updateFilters("B", EIssueFilterType.KANBAN_FILTERS, { group_by: ["done"], sub_group_by: [] });
          expect(filters.getIssueFilters("B", "other")?.kanbanFilters?.group_by).toEqual(["done"]);
          expect(JSON.stringify(filters.getIssueFilters("A", "ws"))).toBe(oldFilters);
          expect(tab.nerve.calls).toEqual([]);
          return;
        }
        const read = await requestedIssue(
          tab.nerve,
          () => actions.fetchIssues("init-loader", { limit: 20, query: { group_by: undefined } }),
          ["GET", "/api/v0/projects/B/issues"]
        );
        read.call.answer(json(200, issuePageOf([archived])));
        await read.promise;
        const list = tab.app.issue.archivedIssues.currentList!;
        expect(list.scope.workspaceSlug).toBe("other");
        expect(list.reconciled.value?.total).toBe(1);
        const sent = await requestedIssue(
          tab.nerve,
          () => (operation === "restore" ? actions.restoreIssue!("B", "old") : actions.removeIssue("B", "old")),
          operation === "restore" ? ["POST", "/api/v0/issues/old/unarchive"] : ["DELETE", "/api/v0/issues/old"]
        );
        sent.call.answer(
          operation === "restore"
            ? json(200, { ...archived, archived_at: null, updated_at: "2026-11-01T00:00:00Z" })
            : noContent()
        );
        await expect(sent.promise).resolves.toBeUndefined();
        expect(list.reconciled.value?.total).toBe(0);
        expect(tab.app.issue.archivedIssues.getIssueIds()).toEqual([]);
        expect(tab.nerve.calls).toHaveLength(2);
      }
    );
  }
);
