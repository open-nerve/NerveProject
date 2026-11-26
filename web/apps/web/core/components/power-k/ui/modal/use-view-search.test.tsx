// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { SWRConfig, type SWRConfiguration } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, Project, View } from "@nerve/api-client";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { useSessionSWR } from "@/lib/use-session-swr";
import { issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProject, stateOf } from "@/store/project/fake-projects";
import { viewOf } from "@/store/views/fake-views";
import { PowerKModalSearchMenu } from "./search-menu";

// The command palette's search names the views of its project (P6a spec 2.8, R13, R20; use-view-search.ts): at a
// project's scope alone, while the caller is offered the ways into them (viewEntryOn: a guest's too), of the store's
// list of the project's views, filtered here by name in any case, in nerve's order. While the store lacks the list,
// the open palette reads it by the views list's key; until nerve answers, or once it refused, the group names nothing
// and holds up no other result. The search menu's other groups are search-menu.test.tsx's.

const ui = vi.hoisted(() => ({
  open: true,
  navigate: vi.fn(),
  close: vi.fn(),
  params: { workspaceSlug: "ws", projectId: "A" as string | undefined },
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "search" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "search" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("react-router", () => ({ useParams: () => ui.params, useNavigate: () => ui.navigate }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof issueStoreTab>>, root: Root, host: HTMLDivElement, config: SWRConfiguration;
let projects: Project[];
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
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
const render = (searchTerm: string, isWorkspaceLevel = false, beside?: ReactNode) =>
  act(async () => {
    root.render(
      <SWRConfig value={config}>
        {beside}
        <Command shouldFilter={false}>
          <Command.List>
            <PowerKModalSearchMenu
              isOpen={ui.open}
              activePage={null}
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
/** nerve's reads of the project's views so far. */
const viewReads = () => tab.nerve.calls.filter((call) => call.path === "/api/v0/projects/A/views");
const answer = (rows: Issue[] = []) =>
  act(async () =>
    searches()
      .at(-1)!
      .answer(json(200, { data: rows }))
  );
const options = () => [...host.querySelectorAll('[role="option"]')].map((element) => element.textContent);
const headings = () => [...host.querySelectorAll("[cmdk-group-heading]")].map((heading) => heading.textContent);
const noResults = () => host.textContent!.includes("power_k.search_menu.no_results");
const searched = async (term: string, isWorkspaceLevel = false) => {
  const asked = searches().length;
  await render(term, isWorkspaceLevel);
  await tick(500);
  // nerve answers the search of work items, which asks it only of a project the caller has a role in
  if (searches().length > asked) await answer();
};
/** The store holds the project's views, as nerve listed them. */
const hold = (views: View[]) =>
  answered(
    tab.nerve,
    () => tab.app.views.fetchProjectViews("A"),
    ["GET", "/api/v0/projects/A/views"],
    { data: views },
    "the views"
  );
/** The views list's read of the project's views (its page.tsx), by its key. */
function ListPageRead() {
  useSessionSWR(["PROJECT_VIEWS", "A"], (id) => tab.app.views.fetchProjectViews(id));
  return null;
}
/** The project as nerve now gives it, as fields say: the store reads it, and the directory lists it so. */
async function becomes(fields: Partial<Project>) {
  const project = { ...tab.project, ...fields };
  projects = [project];
  await loadProject(tab.nerve, tab.app.projectRoot.project, tab.project, project);
}
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
  tab = await issueStoreTab();
  projects = [tab.project];
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states")) {
      const project = tab.app.projectRoot.project.getProjectById(call.path.split("/").at(-2))!;
      call.answer(json(200, { data: [stateOf(project, "Todo", "unstarted", 1, { id: "state" })] }));
    } else if (/workspaces\/[^/]+\/projects$/.test(call.path)) call.answer(json(200, { data: projects }));
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

it("projectScopeNamesTheViewsTheStoreHolds, filtered here by name in any case, in nerve's order, reading none", async () => {
  const bugs = viewOf(tab.project, "Sprint bugs", "u-ann");
  const next = viewOf(tab.project, "Next SPRINT", "u-bob");
  await hold([bugs, viewOf(tab.project, "Triage", "u-ann"), next]);
  const before = viewReads().length;
  await searched("sPrInT");
  expect(viewReads()).toHaveLength(before);
  expect(headings()).toEqual(["views"]);
  expect(options()).toEqual(["A Sprint bugs", "A Next SPRINT"]);
  expect(noResults()).toBe(false);
  await act(async () => (host.querySelectorAll('[role="option"]')[1] as HTMLElement).click());
  expect(ui.navigate.mock.calls).toEqual([[`/ws/projects/A/views/${next.id}`]]);
  expect(ui.close).toHaveBeenCalledOnce();
});

it("readsTheListTheStoreLacksByTheListPagesKey: nerve is asked once beside the list's read, and not again once held", async () => {
  await render("bugs", false, <ListPageRead />);
  await tick(500);
  expect(viewReads()).toHaveLength(1);
  await answer();
  expect(noResults()).toBe(true);
  await act(async () => viewReads()[0]!.answer(json(200, { data: [viewOf(tab.project, "Bugs", "u-ann")] })));
  expect(options()).toEqual(["A Bugs"]);
  expect(noResults()).toBe(false);
  // the palette closed and opened again, long after: the store holds the list, nerve is not asked again
  ui.open = false;
  await render("bugs");
  await tick(5000);
  ui.open = true;
  await render("bugs");
  await tick(500);
  await answer();
  expect(viewReads()).toHaveLength(1);
  expect(options()).toEqual(["A Bugs"]);
});

it("workspaceScopeNamesNoViews, the route's project's neither, and reads none", async () => {
  await hold([viewOf(tab.project, "Bugs", "u-ann")]);
  const before = viewReads().length;
  await searched("bugs", true);
  expect(host.textContent).not.toContain("Bugs");
  expect(noResults()).toBe(true);
  expect(viewReads()).toHaveLength(before);
});

it("names the project's views to its guest, a guest's ways into them his too", async () => {
  await becomes({ member_role: 5 });
  await searched("bugs");
  expect(viewReads()).toHaveLength(1);
  await act(async () => viewReads()[0]!.answer(json(200, { data: [viewOf(tab.project, "Bugs", "u-ann")] })));
  expect(options()).toEqual(["A Bugs"]);
});

it.each([
  ["its views are turned off", { issue_views_view: false }],
  ["the caller is not its member", { member_role: null }],
] as const)("names none of the views the store holds when %s", async (_, fields) => {
  await hold([viewOf(tab.project, "Bugs", "u-ann")]);
  await becomes(fields);
  await searched("bugs");
  expect(tab.app.projectRoot.project.getProjectById("A")).toMatchObject(fields);
  expect(host.textContent).not.toContain("Bugs");
  expect(noResults()).toBe(true);
});

it.each([
  ["its views are turned off", { issue_views_view: false }],
  ["the caller is not its member", { member_role: null }],
] as const)("reads none of the project's views when %s", async (_, fields) => {
  await becomes(fields);
  await searched("bugs");
  expect(viewReads()).toEqual([]);
});

it("a list not answered or refused names no view and holds up no other result", async () => {
  await render("a");
  await tick(500);
  await answer([{ ...issueOf("i1"), name: "Alpha notes", sequence_id: 7 }]);
  expect(viewReads()).toHaveLength(1);
  expect(options()).toEqual(["A-7Alpha notes", "A"]);
  expect(noResults()).toBe(false);
  await act(async () => viewReads()[0]!.answer(problem(403, "forbidden")));
  expect(options()).toEqual(["A-7Alpha notes", "A"]);
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(host.querySelector('[role="status"]')).toBeNull();
});

it("noResultsOnlyWhenEveryGroupIsEmpty: a view alone is a result", async () => {
  await hold([viewOf(tab.project, "Bugs", "u-ann")]);
  await searched("bugs");
  expect(options()).toEqual(["A Bugs"]);
  expect(noResults()).toBe(false);
  await searched("nothing");
  expect(noResults()).toBe(true);
});
