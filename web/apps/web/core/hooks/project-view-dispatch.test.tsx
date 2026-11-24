// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a project's view's page's layouts read and do by the page's store type (P6a spec 2.5, R1; re-96 I3), where a
// store type the list engine missed would say nothing: each view's list is read on its own as the route moves from one
// view to another (the list's mount key names the view); the header's display panel offers the project page's
// options; the layouts take the view page's filters, its collapsed groups with them; a drop goes through the view
// page's commands. The session's stores, hooks, SWR and HTTP are real.
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType, type IIssueFilters } from "@nerve/types";
import { usePageDisplay } from "@/components/issues/use-page-display";
import { StoreContext } from "@/lib/store-context";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import type { RootStore } from "@/store/root.store";
import { viewOf } from "@/store/views/fake-views";
import { usePageIssueFilters } from "./store/use-page-issue-filters";
import { useGroupIssuesDragNDrop } from "./use-group-dragndrop";
import { useIssueListMount } from "./use-issue-list-mount";
import { useIssuesActions } from "./use-issues-actions";

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
let cache = new Map();
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  renderer = createRoot(document.createElement("div"));
  cache = new Map();
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
/** A view of project A as nerve holds it, of the work items of priority, with display. */
const viewOfPriority = (name: string, priority: string, display: Partial<View["display_filters"]> = {}): View => {
  const view = viewOf(tab.project, name, "u-me", { filters: { priority: [priority] } });
  return { ...view, display_filters: { ...view.display_filters, ...display } };
};
/** The route is the view's page, whose draft of the view nerve's answer seeded. */
async function onViewPage(view: View) {
  fixture.params = { workspaceSlug: "ws", projectId: "A", viewId: view.id };
  const read = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.projectViewIssuesFilter.fetchFilters("ws", view.id),
    ["GET", `/api/v0/views/${view.id}`]
  );
  read.call.answer(json(200, view));
  await read.promise;
}
const render = (children: ReactNode) =>
  act(async () =>
    renderer.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => cache }}>{children}</SWRConfig>
      </StoreContext.Provider>
    )
  );

/** A layout of the view page: it reads its list as the layouts do. */
function ViewLayout() {
  const { fetchIssues } = useIssuesActions(EIssuesStoreType.PROJECT_VIEW);
  useIssueListMount(EIssuesStoreType.PROJECT_VIEW, tab.app.issue, fetchIssues, { limit: 20 });
  return null;
}

it("reads a view's own list when the route moves to it while another view's read is out", async () => {
  const urgent = viewOfPriority("Urgent", "urgent");
  const low = viewOfPriority("Low", "low");
  await onViewPage(low);
  await onViewPage(urgent);
  await render(<ViewLayout key={urgent.id} />);
  await until(() => lists().length === 1, "the urgent view's list");
  expect(lists()[0]!.query).toMatchObject({ priority: "urgent" });
  // the page of the other view, mounted anew, as the view page's layout keyed by the view is
  fixture.params = { ...fixture.params, viewId: low.id };
  await render(<ViewLayout key={low.id} />);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(lists(), "the low view's own read").toHaveLength(2);
  expect(lists()[1]!.query).toMatchObject({ priority: "low" });
  await act(async () => {
    lists()[0]!.answer(json(200, issuePageOf([issueOf("u")])));
    lists()[1]!.answer(json(200, issuePageOf([issueOf("l")])));
  });
  await until(() => tab.app.issue.projectViewIssues.getIssueIds()?.[0] === "l", "the low view's list");
});

it("offers the project page's display options in a view's page's header, and its draft's display", async () => {
  const view = viewOfPriority("Board", "urgent", { layout: "kanban", group_by: "state_id" });
  await onViewPage(view);
  let display: ReturnType<typeof usePageDisplay> | undefined;
  const Header = () => {
    display = usePageDisplay(EIssuesStoreType.PROJECT_VIEW, view.id);
    return null;
  };
  await render(<Header />);
  expect(display?.layout).toBe("kanban");
  expect(display?.display.layoutDisplayFiltersOptions).toBe(ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions.kanban);
  expect(display?.display.displayFilters).toMatchObject({ layout: "kanban", group_by: "state" });
});

it("gives a view's page's layouts its draft's filters, the collapsed groups with them", async () => {
  const view = viewOfPriority("Urgent", "urgent");
  await onViewPage(view);
  await tab.app.issue.projectViewIssuesFilter.updateFilters("ws", view.id, EIssueFilterType.KANBAN_FILTERS, {
    group_by: ["done"],
    sub_group_by: [],
  });
  let filters: IIssueFilters | undefined;
  const Layout = () => {
    filters = usePageIssueFilters(EIssuesStoreType.PROJECT_VIEW);
    return null;
  };
  await render(<Layout />);
  expect(filters).toMatchObject({
    richFilters: { and: [{ priority__in: "urgent" }] },
    kanbanFilters: { group_by: ["done"], sub_group_by: [] },
  });
  expect(tab.app.issue.projectIssuesFilter.filters.A, "the project page's").toBeUndefined();
});

it("drops a row of a view's page's list through the view page's commands", async () => {
  const view = viewOfPriority("Ordered", "urgent", { order_by: "sort_order" });
  await onViewPage(view);
  const read = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.projectViewIssues.fetchIssues("ws", "A", view.id, "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  read.call.answer(json(200, issuePageOf([issueOf("x"), { ...issueOf("n"), sort_order: 2000 }])));
  await read.promise;
  let drop!: ReturnType<typeof useGroupIssuesDragNDrop>;
  const Layout = () => {
    drop = useGroupIssuesDragNDrop(EIssuesStoreType.PROJECT_VIEW, "sort_order", undefined);
    return null;
  };
  await render(<Layout />);
  const sent = await requestedIssue(tab.nerve, () => drop({ groupId: "", id: "x" }, { groupId: "", id: "n" }), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  const { sort_order } = sent.call.body as { sort_order: number };
  expect(sent.call.body).toEqual({ sort_order });
  sent.call.answer(json(200, { ...issueOf("x"), sort_order, updated_at: "2026-10-29T00:00:00Z" }));
  await sent.promise;
  expect(tab.app.issue.issues.getIssueById("x")?.sort_order).toBe(sort_order);
});
