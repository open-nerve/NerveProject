// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The header of a cycle's page (P5b spec 2.6, R9; P5a's crumbs, sidebar toggle and quick actions stay): for those who
// see the page's work items, their count as nerve gives it for this cycle, the four layouts, the toggle of the page's
// filters and its display, which the page's own preferences keep, with the cycle page's grouping, never by cycle; a
// new work item of the cycle for those who may create one while the cycle takes them. None of the list's while the
// project's cycles are off. The stores, the work items below and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { ISSUE_GROUP_BY_OPTIONS, ISSUE_STORE_TO_FILTERS_MAP } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { cycleOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  buttonOf,
  C1,
  inHeader,
  planningPageTab,
  cyclePreferencesOf,
  layouts,
} from "@/components/issues/issue-layouts/roots/fake-planning-page";
import { CycleIssuesHeader } from "./header";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; the rows below and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// P5a's crumbs and the cycle's quick actions have their own tests (page.test.tsx, actions-host.test.tsx).
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/cycles/quick-actions", () => ({ CycleQuickActions: () => null }));

const C2 = "019a0000-0000-7000-8000-0000000000c2";
let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the cycle's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.CYCLE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
});
afterEach(async () => page.close());

/** The page's header, of the route's cycle, keyed by it as the page's layout keys it. */
const header = (cycleId = C1) => (
  <div data-header key={cycleId}>
    <CycleIssuesHeader workspaceSlug="ws" projectId="A" cycleId={cycleId} />
  </div>
);
/** The count the header shows: a leaf of only digits. */
const count = () =>
  [...document.querySelectorAll("[data-header] *")].find(
    (node) => node.children.length === 0 && /^\d+$/.test(node.textContent ?? "")
  )?.textContent;
/** What of the page's work items the header offers. */
function offered() {
  return {
    count: count(),
    layouts: ["list", "kanban", "calendar", "spreadsheet"].filter((layout) =>
      inHeader(`issue.layouts.title.${layout}`)
    ),
    filters: !!inHeader("common.filter_options.add"),
    display: !!inHeader("common.display"),
    create: !!inHeader("issue.add.label"),
  };
}
// nerve counts seven work items in the cycle though it sends the first page's one
const seven = { total: 7, groups: [{ ...(layouts.list.page() as { groups: object[] }).groups[0], total: 7 }] };
const all = ["list", "kanban", "calendar", "spreadsheet"];
const ended: Partial<Cycle> = { status: "completed" };
// nerve archives only ended cycles: a status other than completed shows that being archived alone takes the creation
const archived: Partial<Cycle> = { status: "current", archived_at: "2026-12-01T09:00:00Z" };

it.each([
  { who: "an admin", role: 20 as const },
  { who: "a member", role: 15 as const },
])(
  "offers $who of a cycle under way its count, layouts, filters, display and a new work item of it",
  async ({ role }) => {
    await opened({ role });
    await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
    expect(offered()).toEqual({ count: "7", layouts: all, filters: true, display: true, create: true });
    await act(async () => inHeader("issue.add.label")?.click());
    expect(page.app.commandPalette.isCreateIssueModalOpen).toBe(true);
    expect(page.app.commandPalette.createIssueStoreType).toBe(EIssuesStoreType.CYCLE);
  }
);

it("offers a guest the cycle's count, layouts, filters and display, without a new work item", async () => {
  await opened({ role: 5 });
  await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
  expect(offered()).toEqual({ count: "7", layouts: all, filters: true, display: true, create: false });
});

it.each([
  { cycle: "ended", fields: ended },
  { cycle: "archived", fields: archived },
])(
  "offers an admin of a cycle $cycle its count, layouts, filters and display, without a new work item",
  async ({ fields }) => {
    await opened({ role: 20, cycle: fields });
    await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
    expect(offered()).toEqual({ count: "7", layouts: all, filters: true, display: true, create: false });
  }
);

it("offers none of the list's controls while the project's cycles are off, only the page's own", async () => {
  await opened({ role: 20, project: { cycle_view: false } });
  await page.mount(header());
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: false });
  expect(inHeader("aria_labels.toggle_details")).toBeDefined();
  expect(page.nerve.calls).toEqual([]);
});

it("offers none of the list's controls to one who is not the project's member, whom the page lists nothing", async () => {
  await opened({ project: { member_role: null } });
  await page.mount(header());
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: false });
  expect(inHeader("aria_labels.toggle_details"), "the page's own header").toBeDefined();
  expect(page.nerve.calls).toEqual([]);
});

it("offers no layout, filter or display before nerve gives the page's preferences, and offers them once it has", async () => {
  await opened({ role: 15 });
  await page.mount(header());
  await until(() => page.preferences().length === 1, "the page's preferences");
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: true });
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(page.preferences().map(({ method }) => method)).toEqual(["GET"]);
  await act(async () => page.preferences()[0]!.answer(json(200, cyclePreferencesOf(layouts.list.display))));
  expect(offered()).toMatchObject({ layouts: all, filters: true, display: true, create: true });
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await until(() => page.preferences().length === 2, "the change of the page's layout");
  expect(page.preferences()[1]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
});

it("offers no layout, filter or display while nerve could not give the page's preferences, and offers them once it has", async () => {
  await opened({ role: 15 });
  await page.mount(header());
  await until(() => page.preferences().length === 1, "the page's preferences");
  await act(async () => page.preferences()[0]!.answer(problem(503, "server_busy")));
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: true });
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(page.preferences().map(({ method }) => method)).toEqual(["GET"]);
  await act(async () => buttonOf("common.retry")?.click());
  await until(() => page.preferences().length === 2, "the preferences again");
  await act(async () => page.preferences()[1]!.answer(json(200, cyclePreferencesOf(layouts.list.display))));
  expect(offered()).toMatchObject({ layouts: all, filters: true, display: true, create: true });
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await until(() => page.preferences().length === 3, "the change of the page's layout");
  expect(page.preferences()[2]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
});

it("hides the list's controls and its count once the project's cycles are turned off, its list still the store's", async () => {
  await opened({ role: 20 });
  await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
  expect(offered()).toEqual({ count: "7", layouts: all, filters: true, display: true, create: true });
  await act(async () =>
    page.app.projectRoot.project.confirmProject(page.project, (project) => ({ ...project, cycle_view: false }))
  );
  expect(page.app.issue.cycleIssues.pageTotal({ projectID: "A", entityID: C1 })).toBe(7);
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: false });
});

// The route names a cycle the tab holds of another project than the route's (P6b spec 5, review re-71 M2): the page
// does not have it (it says there is no such cycle here), and its header offers no new work item of it.
it("offers no new work item of a cycle the tab holds of another project than the route's", async () => {
  await opened({ role: 15 });
  const other = projectOf("B", page.workspace.id, { id: "B" });
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [page.project, other]);
  expect(page.app.planning.cycles.getCycleById(C1)?.project_id).toBe("A");
  bound.params = { workspaceSlug: "ws", projectId: "B", cycleId: C1 };
  await page.render(
    <MemoryRouter>
      <div data-header>
        <CycleIssuesHeader workspaceSlug="ws" projectId="B" cycleId={C1} />
      </div>
    </MemoryRouter>
  );
  expect(inHeader("aria_labels.toggle_details"), "the header of the route's page").toBeDefined();
  expect(inHeader("issue.add.label")).toBeUndefined();
});

// The same, once the tab has shown the cycle's page in its own project, so that it holds the cycle's list and
// preferences too (P6b spec 5, review re-71 M2, fix F9): the route's project's header shows nothing of the cycle,
// neither its name in the crumb nor its quick actions, nor anything of its list.
it("shows nothing of a cycle of another project than the route's, after its own project's page", async () => {
  const quickActions = vi.spyOn(await import("@/components/cycles/quick-actions"), "CycleQuickActions");
  await opened({ role: 15 });
  await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
  expect(offered()).toEqual({ count: "7", layouts: all, filters: true, display: true, create: true });
  expect(document.querySelector("[data-header]")!.textContent, "the crumb of the cycle").toContain("Now");
  expect(quickActions.mock.calls.map(([props]) => props.cycleId)).toContain(C1);
  const other = projectOf("B", page.workspace.id, { id: "B" });
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [page.project, other]);
  quickActions.mockClear();
  bound.params = { workspaceSlug: "ws", projectId: "B", cycleId: C1 };
  await page.render(
    <MemoryRouter>
      <div data-header>
        <CycleIssuesHeader workspaceSlug="ws" projectId="B" cycleId={C1} />
      </div>
    </MemoryRouter>
  );
  expect(inHeader("aria_labels.toggle_details"), "the header of the route's page").toBeDefined();
  expect(offered()).toEqual({ count: undefined, layouts: [], filters: false, display: false, create: false });
  expect(document.querySelector("[data-header]")!.textContent, "the crumb").not.toContain("Now");
  expect(quickActions, "the cycle's quick actions").not.toHaveBeenCalled();
});

it("shows no count of the cycle before, on the page of another whose list nerve has not given", async () => {
  await opened({ role: 20 });
  await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
  expect(count()).toBe("7");
  const next = cycleOf(page.project, "Next", { id: C2, status: "current" });
  await answered(
    page.nerve,
    () => page.app.planning.cycles.fetchCycle(C2),
    ["GET", `/api/v0/cycles/${C2}`],
    next,
    "C2"
  );
  bound.params = { ...bound.params, cycleId: C2 };
  await page.mount(header(C2));
  await until(() => page.nerve.to(`/api/v0/me/cycles/${C2}/preferences`).length === 1, "the next page's preferences");
  expect(page.app.issue.cycleIssues.total, "the cycle before's list, still the store's").toBe(7);
  expect(count()).toBeUndefined();
});

it("changes the layout and the display by the page's own preferences, never the project's", async () => {
  await opened({ role: 15 });
  await page.show(cyclePreferencesOf(layouts.list.display), seven, header());
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await until(() => page.preferences().length === 2, "the change of the page's layout");
  expect(page.preferences()[1]).toMatchObject({ method: "PATCH" });
  expect(page.preferences()[1]!.body).toMatchObject({ display_filters: { layout: "kanban" } });
  await act(async () => page.preferences()[1]!.answer(json(200, cyclePreferencesOf(layouts.kanban.display))));
  await act(async () => inHeader("common.display")?.click());
  await act(async () => inHeader("common.priority")?.click());
  await until(() => page.preferences().length === 3, "the change of the page's display");
  expect(page.preferences()[2]).toMatchObject({ method: "PATCH" });
  expect(page.preferences()[2]!.body).toMatchObject({ display_properties: { priority: false } });
  expect(page.nerve.to("/api/v0/me/projects/A/preferences")).toEqual([]);
});

it.each(["list", "kanban"] as const)(
  "groups the cycle's %s by the cycle page's options, never by a cycle (R9)",
  async (layout) => {
    const options = ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.CYCLE]!.layoutOptions[layout]!.display_filters;
    const groupings = [...(options.group_by ?? []), ...(options.sub_group_by ?? [])];
    expect(groupings.length).toBeGreaterThan(0);
    expect(groupings.filter((option) => /cycle/i.test(String(option)))).toEqual([]);
    await opened({ role: 15 });
    await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page(), header());
    await act(async () => inHeader("common.display")?.click());
    const shown = [...document.querySelectorAll("[data-header] button[aria-pressed]")].map((node) => node.textContent);
    for (const option of ISSUE_GROUP_BY_OPTIONS.filter(({ key }) => options.group_by?.includes(key)))
      expect(shown, option.titleTranslationKey).toContain(option.titleTranslationKey);
    expect(shown.filter((title) => /cycle/i.test(title ?? ""))).toEqual([]);
  }
);
