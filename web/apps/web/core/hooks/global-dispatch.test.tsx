// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a workspace's view's page reads and does by its store type, GLOBAL (M7/P6b spec 2.3, R1, R8; re-97 m1), where a
// dispatch point that missed it would say nothing: the route names it; its list and preferences are the workspace's
// pages' own; its commands are the workspace's, every one a list page has save a calendar's and a drag's, a change of
// its display the route page's, followed by a read of its list; its filters are the route page's; its header's display
// panel offers the spreadsheet's options whatever layout a saved view holds, without a cycle or the modules, which its
// table never shows. The session's stores, hooks and HTTP are real.
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType, type IIssueFilters } from "@nerve/types";
import { usePageDisplay } from "@/components/issues/use-page-display";
import { FilterDisplayProperties } from "@/components/issues/issue-layouts/filters/header/display-filters/display-properties";
import { StoreContext } from "@/lib/store-context";
import { json } from "@/lib/auth/fake-nerve";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import type { RootStore } from "@/store/root.store";
import { workspaceViewOf } from "@/store/views/fake-views";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { usePageIssueFilters } from "./store/use-page-issue-filters";
import { useIssues } from "./store/use-issues";
import { useIssueStoreType } from "./use-issue-layout-store";
import { useIssuesActions, type IssueActions } from "./use-issues-actions";

const fixture = vi.hoisted(() => ({ params: {} as Record<string, string>, app: undefined as RootStore | undefined }));
vi.mock("react-router", () => ({ useParams: () => fixture.params }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
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
let renderer: ReturnType<typeof createRoot>;
let host: HTMLDivElement;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.params = { workspaceSlug: "ws", globalViewId: "assigned" };
  host = document.createElement("div");
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const LIST: [string, string] = ["GET", "/api/v0/workspaces/ws/issues"];
const ASSIGNED = workspacePageId("ws", "assigned");
const render = (children: ReactNode) =>
  act(async () => renderer.render(<StoreContext.Provider value={tab.app}>{children}</StoreContext.Provider>));
/** The route's page's hooks, as a layout of the page calls them, their values the test's. */
async function onPage<T>(use: () => T): Promise<() => T> {
  let value!: T;
  const Probe = () => {
    value = use();
    return null;
  };
  await render(<Probe />);
  return () => value;
}

it("names GLOBAL on a workspace's view's route, and gives the workspace's pages' list and preferences", async () => {
  const got = await onPage(() => ({ type: useIssueStoreType(), store: useIssues(EIssuesStoreType.GLOBAL) }));
  expect(got().type).toBe(EIssuesStoreType.GLOBAL);
  expect(got().store.issues).toBe(tab.app.issue.globalIssues);
  expect(got().store.issuesFilter).toBe(tab.app.issue.globalIssuesFilter);
});

it("gives every command a list page has, save a calendar's and a drag's, the list read for the route's page", async () => {
  await tab.app.issue.globalIssuesFilter.fetchFilters("ws", ASSIGNED);
  const actions = await onPage(() => useIssuesActions(EIssuesStoreType.GLOBAL));
  expect(Object.keys(actions()).toSorted()).toEqual(
    [
      "archiveIssue",
      "createIssue",
      "createIssueAsGiven",
      "fetchIssues",
      "fetchNextIssues",
      "removeIssue",
      "updateFilters",
      "updateIssue",
    ].toSorted()
  );
  const read = await requestedIssue(tab.nerve, () => actions().fetchIssues("init-loader", { limit: 100 }), LIST);
  expect(read.call.query).toMatchObject({ assignee_id: "me", limit: "100" });
  read.call.answer(json(200, issuePageOf([issueOf("x")])));
  await read.promise;
  expect(tab.app.issue.globalIssues.getIssueIds()).toEqual(["x"]);
});

it("reads the list for the workspace's guest too", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspaceOf("ws", { role: 5 })]);
  tab.nerve.calls.length = 0;
  await tab.app.issue.globalIssuesFilter.fetchFilters("ws", ASSIGNED);
  const actions = await onPage(() => useIssuesActions(EIssuesStoreType.GLOBAL));
  const read = await requestedIssue(tab.nerve, () => actions().fetchIssues("init-loader", { limit: 100 }), LIST);
  read.call.answer(json(200, issuePageOf([])));
  await read.promise;
});

it("changes the route page's display whatever id its caller names, and reads the list again with it", async () => {
  await tab.app.issue.globalIssuesFilter.fetchFilters("ws", ASSIGNED);
  let actions: IssueActions | undefined;
  await onPage(() => (actions = useIssuesActions(EIssuesStoreType.GLOBAL)));
  const first = await requestedIssue(tab.nerve, () => actions!.fetchIssues("init-loader", { limit: 100 }), LIST);
  first.call.answer(json(200, issuePageOf([issueOf("x")])));
  await first.promise;
  // the spreadsheet names the route's project, none on a workspace's route
  const again = await requestedIssue(
    tab.nerve,
    () => actions!.updateFilters("", EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" }),
    LIST
  );
  expect(again.call.query).toMatchObject({ order_by: "priority", assignee_id: "me" });
  again.call.answer(json(200, issuePageOf([issueOf("x")])));
  await again.promise;
  expect(tab.app.issue.globalIssuesFilter.getIssueFilters(ASSIGNED)?.displayFilters?.order_by).toBe("priority");
  expect(tab.app.issue.globalIssuesFilter.filters[""]).toBeUndefined();
});

it("gives the layouts the route page's filters", async () => {
  await tab.app.issue.globalIssuesFilter.fetchFilters("ws", ASSIGNED);
  await tab.app.issue.globalIssuesFilter.updateFilterExpression("ws", ASSIGNED, { and: [{ priority__in: "high" }] });
  let filters: IIssueFilters | undefined;
  await onPage(() => (filters = usePageIssueFilters(EIssuesStoreType.GLOBAL)));
  expect(filters?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  fixture.params = { workspaceSlug: "ws", globalViewId: "created" };
  await onPage(() => (filters = usePageIssueFilters(EIssuesStoreType.GLOBAL)));
  expect(filters, "another page's").toBeUndefined();
});

it("offers the spreadsheet's display options for a view held as a list, without a cycle or the modules", async () => {
  const view: View = workspaceViewOf(tab.workspace, "Listed", "actor");
  fixture.params = { workspaceSlug: "ws", globalViewId: view.id };
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.globalIssuesFilter.fetchFilters("ws", view.id), [
    "GET",
    `/api/v0/views/${view.id}`,
  ]);
  read.call.answer(json(200, { ...view, display_filters: { ...view.display_filters, layout: "list" } }));
  await read.promise;
  let display: ReturnType<typeof usePageDisplay> | undefined;
  const Header = () => {
    display = usePageDisplay(EIssuesStoreType.GLOBAL, view.id);
    const options = display.display;
    return (
      <FilterDisplayProperties
        displayProperties={options.displayProperties}
        displayPropertiesToRender={options.layoutDisplayFiltersOptions?.display_properties ?? []}
        handleUpdate={options.handleDisplayPropertiesUpdate}
      />
    );
  };
  await render(<Header />);
  expect(display?.layout, "the layout held").toBe("list");
  const spreadsheet = ISSUE_DISPLAY_FILTERS_BY_PAGE.workspace_issues.layoutOptions.spreadsheet;
  expect(display?.display.layoutDisplayFiltersOptions).toBe(spreadsheet);
  const offered = [...host.querySelectorAll("button[aria-pressed]")].map((button) => button.textContent);
  expect(offered).toContain("common.assignee");
  expect(offered).not.toContain("common.cycle");
  expect(offered).not.toContain("common.modules");
  expect(offered).toHaveLength(12);
});
