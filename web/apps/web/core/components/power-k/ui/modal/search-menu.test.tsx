// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { SWRConfig, type SWRConfiguration } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { json, problem } from "@/lib/auth/fake-nerve";
import { PowerKModalSearchMenu } from "./search-menu";
const ui = vi.hoisted(() => ({
  open: true,
  navigate: vi.fn(),
  close: vi.fn(),
  legacy: vi.fn(),
  params: { workspaceSlug: "ws", projectId: "A" as string | undefined },
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "search" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "search" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-power-k", () => ({
  usePowerK: () => ({ isPowerKModalOpen: ui.open, togglePowerKModal: ui.close }),
}));
vi.mock("react-router", () => ({ useParams: () => ui.params, useNavigate: () => ui.navigate }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// Retired transport is isolated solely to make any accidental call observable, never to fake the native API.
vi.mock("@/services/workspace.service", () => ({
  WorkspaceService: class {
    searchWorkspace = ui.legacy;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>, root: Root, host: HTMLDivElement, config: SWRConfiguration;
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
let projects: Project[], holdDirectory: boolean;
const context = (): TPowerKContext => ({
  params: ui.params,
  activeCommand: null,
  activeContext: null,
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: ui.navigate,
  closePalette: ui.close,
  setActiveCommand: () => {},
  setActivePage: () => {},
});
const render = (searchTerm = "", isWorkspaceLevel = false, activePage: "update-theme" | null = null) =>
  act(async () => {
    root.render(
      <SWRConfig value={config}>
        <Command shouldFilter={false}>
          <Command.List>
            <PowerKModalSearchMenu
              activePage={activePage}
              context={context()}
              isWorkspaceLevel={isWorkspaceLevel}
              searchTerm={searchTerm}
              updateSearchTerm={() => {}}
            />
          </Command.List>
        </Command>
      </SWRConfig>
    );
  });
const tick = (ms = 0) => act(async () => vi.advanceTimersByTimeAsync(ms));
const searches = () => tab.nerve.calls.filter((call) => call.path.endsWith("/issue-search"));
const directory = () => tab.nerve.calls.filter((call) => /workspaces\/[^/]+\/projects$/.test(call.path));
const answer = (rows = [issueOf("row")], index = searches().length - 1) =>
  act(async () => searches()[index]!.answer(json(200, { data: rows })));
const options = () => [...host.querySelectorAll('[role="option"]')].map((element) => element.textContent);
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // oxlint-disable-next-line no-extend-native -- jsdom has no scrolling; real cmdk keyboard/scroll behavior is verified in I18 pages.
  HTMLElement.prototype.scrollIntoView = vi.fn();
  ui.open = true;
  ui.params = { workspaceSlug: "ws", projectId: "A" };
  ui.navigate.mockReset();
  ui.close.mockReset();
  ui.legacy.mockReset();
  ui.legacy.mockResolvedValue({
    results: { workspace: [], project: [], issue: [], cycle: [], module: [], issue_view: [] },
  });
  tab = await issueStoreTab();
  projects = [tab.project];
  holdDirectory = false;
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states")) {
      const project = tab.app.projectRoot.project.getProjectById(call.path.split("/").at(-2))!;
      call.answer(json(200, { data: [stateOf(project, "Todo", "unstarted", 1, { id: "state" })] }));
    } else if (/workspaces\/[^/]+\/projects$/.test(call.path) && !holdDirectory)
      call.answer(json(200, { data: projects }));
  };
  config = { provider: () => new Map(), dedupingInterval: 0 };
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  // oxlint-disable-next-line no-extend-native -- restore the exact pre-test descriptor after the jsdom-only capability stub.
  if (scrollDescriptor) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", scrollDescriptor);
  else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.restoreAllMocks();
  vi.useRealTimers();
});
it("no query, a closed palette and property pages never request issues; the directory is shared across queries", async () => {
  ui.open = false;
  await render();
  await tick(500);
  expect(directory()).toHaveLength(0);
  expect(searches()).toHaveLength(0);
  ui.open = true;
  await render();
  await tick();
  expect(directory()).toHaveLength(1);
  expect(searches()).toHaveLength(0);
  await render("中文");
  await tick(500);
  expect(searches()).toHaveLength(1);
  expect(directory()).toHaveLength(1);
  await render("A-1");
  await tick(500);
  expect(searches()).toHaveLength(2);
  expect(directory()).toHaveLength(1);
  await render("dark", false, "update-theme");
  await tick(500);
  expect(searches()).toHaveLength(2);
  expect(ui.legacy).not.toHaveBeenCalled();
});
it.each(["中文", "A-1", "1", "%_\\"])(
  "one debounce passes the exact native query %s without legacy transport",
  async (q) => {
    await render(q);
    await tick(499);
    expect(searches()).toHaveLength(0);
    await tick(1);
    expect(searches()).toHaveLength(1);
    expect(searches()[0]!.query).toEqual({ q, project_id: "A" });
    expect(ui.legacy).not.toHaveBeenCalled();
  }
);
it("scopeSwitchOmitsProjectAndDropsOldResultsImmediately", async () => {
  await render("first");
  await tick(500);
  expect(searches()).toHaveLength(1);
  await answer([issueOf("first")]);
  expect(host.textContent).toContain("first");
  await render("first", true);
  expect(host.textContent).not.toContain("A-1first");
  expect(options()).toEqual([]);
  await tick(500);
  expect(searches()[1]!.query).toEqual({ q: "first" });
  await answer([issueOf("workspace")]);
  expect(host.textContent).toContain("workspace");
  await render("new", true);
  expect(options()).toEqual([]);
  await tick(500);
  await render("last", true);
  await answer([issueOf("late")]);
  expect(options()).toEqual([]);
  expect(tab.app.issue.issues.held("late")).toBeUndefined();
});
it("nativeSearchOrderSurvivesCommandFiltering and selection uses native identifiers", async () => {
  await render("relevance");
  await tick(500);
  expect(searches()).toHaveLength(1);
  await answer([
    { ...issueOf("z"), name: "First relevance", sequence_id: 3 },
    { ...issueOf("a"), name: "Second relevance", sequence_id: 9 },
  ]);
  expect(options()).toEqual(["A-3First relevance", "A-9Second relevance"]);
  await act(async () => (host.querySelector('[role="option"]') as HTMLElement).click());
  expect(ui.navigate).toHaveBeenCalledExactlyOnceWith("/ws/browse/A-3");
  expect(ui.close).toHaveBeenCalledWith(false);
});
it.each([403, 404, 500])("searchRefusalIsNotNoResults: %s is shown and retries only GET", async (status) => {
  await render("problem");
  await tick(500);
  expect(searches()).toHaveLength(1);
  await act(async () => searches()[0]!.answer(problem(status, "forbidden")));
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.forbidden");
  expect(host.textContent).not.toContain("power_k.search_menu.no_results");
  const retry = [...host.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!;
  await act(async () => retry.click());
  await tick(500);
  expect(searches()).toHaveLength(2);
  await answer();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("localProjectsUseRouteWorkspace, exclude archived and wait for the directory rather than claiming no results", async () => {
  const other = workspaceOf("other");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  projects = [
    projectOf("Beta", other.id, { id: "B" }),
    projectOf("Beta-old", other.id, { id: "old", archived_at: "2026-10-31T00:00:00Z" }),
  ];
  ui.params = { workspaceSlug: "other", projectId: undefined };
  holdDirectory = true;
  await render("beta", true);
  await tick(500);
  expect(searches()).toHaveLength(1);
  await answer([]);
  expect(host.textContent).not.toContain("power_k.search_menu.no_results");
  await act(async () => directory()[0]!.answer(problem(500, "internal_error")));
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.internal_error");
  holdDirectory = false;
  await act(async () => [...host.querySelectorAll("button")].find((b) => b.textContent === "common.retry")!.click());
  await tick();
  expect(options()).toEqual(["Beta"]);
  expect(tab.app.router.workspaceSlug).toBe("ws");
  await act(async () => (host.querySelector('[role="option"]') as HTMLElement).click());
  expect(ui.navigate).toHaveBeenCalledExactlyOnceWith("/other/projects/B/issues");
});
it("closing or retiring the root immediately hides loaded rows", async () => {
  await render("row");
  await tick(500);
  expect(searches()).toHaveLength(1);
  await answer();
  expect(options()).toHaveLength(1);
  ui.open = false;
  await render("row");
  expect(options()).toEqual([]);
  ui.open = true;
  await render("row");
  await tick(500);
  await answer();
  expect(options()).toHaveLength(1);
  await act(async () => tab.app.issue.retire());
  expect(options()).toEqual([]);
});
