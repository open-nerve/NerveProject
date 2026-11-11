// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ProjectsAppPowerKProvider } from "./projects-app-provider";
import { PowerKStore } from "@/store/power-k.store";

const ui = vi.hoisted(() => ({ toggle: vi.fn(), setPage: vi.fn(), navigate: vi.fn() }));
type Params = { workspaceSlug?: string; projectId?: string; workItem?: string };
const ParamsContext = createContext<Params>({});
const PowerKContext = createContext<PowerKStore | undefined>(undefined);
vi.mock("react-router", () => ({ useParams: () => useContext(ParamsContext), useNavigate: () => ui.navigate }));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({
    issue: {
      getIssueById: () => undefined,
      getIssueIdByIdentifier: () => undefined,
    },
  }),
}));
vi.mock("@/hooks/store/use-power-k", () => ({
  usePowerK: () =>
    useContext(PowerKContext) ?? {
      activeContext: "global",
      isPowerKModalOpen: false,
      togglePowerKModal: ui.toggle,
      setActivePage: ui.setPage,
    },
}));
vi.mock("./config/commands", () => ({ useProjectsAppPowerKCommands: () => [] }));
vi.mock("./global-shortcuts", () => ({ GlobalShortcutsProvider: () => <span data-shortcuts /> }));
vi.mock("./ui/modal/commands-list", () => ({ ProjectsAppPowerKCommandsList: () => null }));
vi.mock("./ui/modal/wrapper", () => ({ ProjectsAppPowerKModalWrapper: () => <span data-palette /> }));
// The project's dialogs' host says when it mounts, and for which project, and when it unmounts.
const hosts = vi.hoisted((): string[] => []);
vi.mock("@/components/modals/project-level", async () => {
  const { useLayoutEffect, useState } = await import("react");
  return {
    ProjectLevelModals: ({ projectId }: { projectId: string }) => {
      const [mountedFor] = useState(projectId);
      useLayoutEffect(() => {
        hosts.push(`mounted for ${mountedFor}`);
        return () => void hosts.push(`unmounted from ${mountedFor}`);
      }, [mountedFor]);
      return <span data-project />;
    },
  };
});
vi.mock("@/components/modals/workspace-level", () => ({ WorkspaceLevelModals: () => <span data-workspace /> }));
vi.mock("@/components/modals/work-item-level", () => ({ WorkItemLevelModals: () => <span data-work-item /> }));

let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.clearAllMocks();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
const render = (params: Params, powerK?: PowerKStore) =>
  act(async () =>
    root.render(
      <PowerKContext.Provider value={powerK}>
        <ParamsContext.Provider value={params}>
          <ProjectsAppPowerKProvider />
        </ParamsContext.Provider>
      </PowerKContext.Provider>
    )
  );

it("personal settings keeps its palette and shortcuts without mounting work-item forms", async () => {
  await render({});
  expect(host.querySelector("[data-shortcuts]")).not.toBeNull();
  expect(host.querySelector("[data-palette]")).not.toBeNull();
  expect(host.querySelector("[data-work-item]")).toBeNull();
  expect(host.querySelector("[data-project]")).toBeNull();
  expect(host.querySelector("[data-workspace]")).toBeNull();
});

it.each([
  [
    { workspaceSlug: "ws", projectId: "A" },
    { workspaceSlug: "ws", projectId: "B" },
  ],
  [
    { workspaceSlug: "ws", workItem: "A-1" },
    { workspaceSlug: "ws", workItem: "A-2" },
  ],
  [
    { workspaceSlug: "ws", workItem: "A-1" },
    { workspaceSlug: "ws", projectId: "A" },
  ],
])("route scope changes retire the real palette state and Back cannot reopen it: %j", async (first, next) => {
  const powerK = new PowerKStore();
  await render(first, powerK);
  await act(async () => {
    powerK.togglePowerKModal(true);
    powerK.setActivePage("update-work-item-state");
  });
  await render({ ...first }, powerK);
  expect(powerK.isPowerKModalOpen).toBe(true);
  await render(next, powerK);
  expect(powerK.isPowerKModalOpen).toBe(false);
  expect(powerK.activePage).toBeNull();
  await render(first, powerK);
  expect(powerK.isPowerKModalOpen).toBe(false);
});
it("workspace and project routes retain their command hosts and retire them on returning to personal settings", async () => {
  await render({ workspaceSlug: "ws" });
  expect(host.querySelector("[data-work-item]")).not.toBeNull();
  expect(host.querySelector("[data-workspace]")).not.toBeNull();
  await render({ workspaceSlug: "ws", projectId: "project" });
  expect(host.querySelector("[data-project]")).not.toBeNull();
  await render({});
  expect(host.querySelector("[data-work-item]")).toBeNull();
  await act(async () => root.render(null));
  expect(ui.toggle).toHaveBeenLastCalledWith(false);
  expect(ui.setPage).toHaveBeenLastCalledWith(null);
});
it("mounts the project's dialogs' host anew for each project, the host before closing what it renders as it goes", async () => {
  await render({ workspaceSlug: "ws", projectId: "A" });
  hosts.length = 0;
  await render({ workspaceSlug: "ws", projectId: "A" });
  await render({ workspaceSlug: "ws", projectId: "B" });
  expect(hosts).toEqual(["unmounted from A", "mounted for B"]);
});
