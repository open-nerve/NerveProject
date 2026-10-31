// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ProjectsAppPowerKProvider } from "./projects-app-provider";

const ui = vi.hoisted(() => ({ toggle: vi.fn(), setPage: vi.fn(), navigate: vi.fn() }));
type Params = { workspaceSlug?: string; projectId?: string; workItem?: string };
const ParamsContext = createContext<Params>({});
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
  usePowerK: () => ({
    activeContext: "global",
    isPowerKModalOpen: false,
    togglePowerKModal: ui.toggle,
    setActivePage: ui.setPage,
  }),
}));
vi.mock("./config/commands", () => ({ useProjectsAppPowerKCommands: () => [] }));
vi.mock("./global-shortcuts", () => ({ GlobalShortcutsProvider: () => <span data-shortcuts /> }));
vi.mock("./ui/modal/commands-list", () => ({ ProjectsAppPowerKCommandsList: () => null }));
vi.mock("./ui/modal/wrapper", () => ({ ProjectsAppPowerKModalWrapper: () => <span data-palette /> }));
vi.mock("@/components/modals/project-level", () => ({ ProjectLevelModals: () => <span data-project /> }));
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
const render = (params: Params) =>
  act(async () =>
    root.render(
      <ParamsContext.Provider value={params}>
        <ProjectsAppPowerKProvider />
      </ParamsContext.Provider>
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
