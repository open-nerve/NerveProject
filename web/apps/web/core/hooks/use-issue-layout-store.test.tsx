// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { IssuesStoreContext, useIssueStoreType } from "./use-issue-layout-store";
import { useIssuesActions } from "./use-issues-actions";

const fixture = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  created: [] as { projectId: string; name: string }[],
  fetches: 0,
}));
vi.mock("react-router", () => ({ useParams: () => fixture.params }));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  const issues = {
    fetchIssues: () => {
      fixture.fetches++;
      throw new Error("unexpected list fetch");
    },
    createIssue: async (_workspace: string, projectId: string, data: { name: string }) => {
      const created = { projectId, name: data.name };
      fixture.created.push(created);
      return created;
    },
  };
  return {
    StoreContext: createContext({
      issue: {
        issues: { issuesMap: {} },
        projectIssues: issues,
        projectIssuesFilter: {},
        profileIssues: {},
        profileIssuesFilter: {},
        archivedIssues: {},
        archivedIssuesFilter: {},
        workspaceDraftIssues: {},
      },
    }),
  };
});

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  fixture.params = {};
  fixture.created = [];
  fixture.fetches = 0;
});

async function mount(run: () => void, context?: EIssuesStoreType) {
  const root = createRoot(document.createElement("div"));
  const Probe = () => {
    run();
    return null;
  };
  try {
    await act(async () =>
      root.render(
        <IssuesStoreContext.Provider value={context}>
          <Probe />
        </IssuesStoreContext.Provider>
      )
    );
  } finally {
    await act(async () => root.unmount());
  }
}

it.each(["globalViewId", "viewId", "cycleId", "moduleId"])(
  "uses generic commands on %s routes and creates in the selected project without fetching a list",
  async (route) => {
    fixture.params = {
      workspaceSlug: "acme",
      ...(route === "globalViewId" ? {} : { projectId: "route-project" }),
      [route]: "domain-id",
    };
    let created: unknown;
    await mount(() => {
      const selected = useIssueStoreType();
      expect(selected).toBe(EIssuesStoreType.DEFAULT);
      const actions = useIssuesActions(selected);
      created = actions.createIssue?.("selected-project", { name: "From global create" });
    });
    await expect(created).resolves.toEqual({ projectId: "selected-project", name: "From global create" });
    expect(fixture.created).toEqual([{ projectId: "selected-project", name: "From global create" }]);
    expect(fixture.fetches).toBe(0);
  }
);

it("keeps project/profile route selection, explicit context and a generic route without project", async () => {
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.DEFAULT));
  fixture.params = { projectId: "project" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.PROJECT));
  fixture.params = { userId: "user" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.PROFILE));
  fixture.params = { globalViewId: "view" };
  await mount(() => expect(useIssueStoreType()).toBe(EIssuesStoreType.ARCHIVED), EIssuesStoreType.ARCHIVED);
});

it.each(["GLOBAL", "CYCLE", "MODULE", "PROJECT_VIEW", "unknown"])(
  "rejects %s actions instead of silently dispatching project commands",
  async (kind) => {
    await mount(() => expect(() => useIssuesActions(kind as EIssuesStoreType)).toThrow("Unsupported issue store"));
    expect(fixture.created).toEqual([]);
    expect(fixture.fetches).toBe(0);
  }
);
