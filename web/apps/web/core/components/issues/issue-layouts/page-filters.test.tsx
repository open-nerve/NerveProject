// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The four layouts read the filters of the page in the route by its store type (P5b spec 2.5, re-66 H2): a cycle's page
// reads its own preferences by the route's cycle. The router store names project A and another cycle, whose preferences
// differ from the page's, so a layout that read the router-driven filters would show theirs.
import { act, createElement, type ReactNode } from "react";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CyclePreferences, IssueListDisplayFilters, ProjectPreferences } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { listIssue } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import { BaseCalendarRoot } from "./calendar/base-calendar-root";
import { BaseKanBanRoot } from "./kanban/base-kanban-root";
import { BaseListRoot } from "./list/base-list-root";
import { BaseSpreadsheetRoot } from "./spreadsheet/base-spreadsheet-root";

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
// jsdom has no viewport geometry; rows, cards, days, menus and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

/** The route's cycle, whose page is shown, and the router store's, whose preferences a router-driven read would use. */
const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const first = listIssue("first", 1000, { sequence_id: 9, cycle_id: C1, priority: "urgent", target_date: "2026-12-31" });
const NoActions = () => null;
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const days = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].map((node) =>
    node.getAttribute("aria-label")
  );
const button = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.setSystemTime(new Date(2026, 11, 31, 12));
  tab.app.issue.issueCalendarView.updateCalendarFilters({
    activeMonthDate: new Date(2026, 11, 31),
    activeWeekDate: new Date(2026, 11, 31),
  });
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", cycleId: C2 }));
});
afterEach(async () => tab.close());

type Display = Partial<IssueListDisplayFilters>;
type Properties = Partial<ProjectPreferences["display_properties"]>;
/** A page's preferences as nerve answers them: the list defaults, with the page's own display. */
function pageOf(display: Display = {}, properties: Properties = {}): CyclePreferences {
  const { filters, display_filters, display_properties } = preferencesOf();
  return {
    filters,
    display_filters: { ...display_filters, ...display },
    display_properties: { ...display_properties, ...properties },
  };
}
/**
 * Loads the router's project and cycle with the list defaults and the shown page with its own; the shown page is the
 * route's cycle, or the project when the route names no cycle.
 */
async function load(page: CyclePreferences) {
  const { app, nerve } = tab;
  const cycle = !!bound.params.cycleId;
  const project = cycle ? preferencesOf() : { ...preferencesOf(), ...page };
  await answered(
    nerve,
    () => app.issue.projectIssuesFilter.fetchFilters("ws", "A"),
    ["GET", "/api/v0/me/projects/A/preferences"],
    project,
    "the router's project"
  );
  await answered(
    nerve,
    () => app.issue.cycleIssuesFilter.fetchFilters("ws", C2),
    ["GET", `/api/v0/me/cycles/${C2}/preferences`],
    pageOf(),
    "the router's cycle"
  );
  if (cycle)
    await answered(
      nerve,
      () => app.issue.cycleIssuesFilter.fetchFilters("ws", C1),
      ["GET", `/api/v0/me/cycles/${C1}/preferences`],
      page,
      "the route's cycle"
    );
  nerve.calls.length = 0;
}
async function mount(root: ReactNode) {
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>{root}</MemoryRouter>
    </SWRConfig>
  );
  await until(() => lists().length === 1, "the page's list");
  expect(lists()[0]!.query.cycle_id).toBe(bound.params.cycleId);
  return lists()[0]!;
}
const week = (show_weekends: boolean) => ({ layout: "calendar", calendar: { layout: "week" as const, show_weekends } });
const dated = { total: 1, groups: [{ key: "2026-12-31", total: 1, data: [first], next_cursor: null }] };

it("the list groups by and shows the properties of the cycle page's preferences", async () => {
  await load(pageOf({ group_by: "priority" }, { key: false }));
  const read = await mount(<BaseListRoot QuickActions={NoActions} />);
  expect(read.query).toMatchObject({ group_by: "priority", limit: "50" });
  await act(async () =>
    read.answer(json(200, { total: 1, groups: [{ key: "urgent", total: 1, data: [first], next_cursor: null }] }))
  );
  expect(document.querySelector('a[id^="issue-first"]')).not.toBeNull();
  expect(document.body.textContent).not.toContain("A-9");
});

it("the board's columns and swimlanes are the cycle page's", async () => {
  await load(pageOf({ layout: "kanban", group_by: "priority", sub_group_by: "state_id" }));
  const read = await mount(<BaseKanBanRoot QuickActions={NoActions} />);
  expect(read.query).toMatchObject({ group_by: "priority", sub_group_by: "state_id", limit: "10" });
});

it("the table shows the columns and sort of the cycle page's preferences", async () => {
  const only = Object.fromEntries(Object.keys(pageOf().display_properties).map((key) => [key, key === "priority"]));
  await load(pageOf({ layout: "spreadsheet", order_by: "priority" }, only));
  const read = await mount(<BaseSpreadsheetRoot QuickActions={NoActions} />);
  await act(async () => read.answer(json(200, { total: 1, data: [first], next_cursor: null })));
  expect([...document.querySelectorAll("thead th")].length).toBe(2);
  expect(document.querySelector("th[aria-sort]")?.getAttribute("aria-sort")).toBe("ascending");
});

it("the calendar shows the cycle page's week, weekends and card properties", async () => {
  await load(pageOf(week(true), { key: false }));
  const read = await mount(<BaseCalendarRoot QuickActions={NoActions} />);
  expect(read.query).toMatchObject({ group_by: "target_date", target_date: "2026-12-27..2027-01-02", limit: "30" });
  await act(async () => read.answer(json(200, dated)));
  expect(days()).toEqual([
    "2026-12-27",
    "2026-12-28",
    "2026-12-29",
    "2026-12-30",
    "2026-12-31",
    "2027-01-01",
    "2027-01-02",
  ]);
  expect(document.querySelector(".min-h-\\[5rem\\]")).toBeNull();
  expect(button("Dec 2026 - Jan 2027")?.disabled).toBe(true);
  expect(document.querySelectorAll('a[id="issue-first"]').length).toBeGreaterThan(0);
  expect(document.body.textContent).not.toContain("A-9");
  await act(async () =>
    document.querySelector<HTMLButtonElement>('button[aria-label="issue.calendar.next_period"]')!.click()
  );
  await until(() => lists().length === 2, "the next week");
  expect(lists()[1]!.query).toMatchObject({ target_date: "2027-01-03..2027-01-09", limit: "30", cycle_id: C1 });
});

it.each(["cycle", "project"] as const)(
  "the %s page's calendar options change its own preferences and nothing else",
  async (shown) => {
    if (shown === "project") bound.params = { workspaceSlug: "ws", projectId: "A" };
    const page = pageOf(week(true));
    await load(page);
    const path = shown === "cycle" ? `/api/v0/me/cycles/${C1}/preferences` : "/api/v0/me/projects/A/preferences";
    const read = await mount(<BaseCalendarRoot QuickActions={NoActions} />);
    await act(async () => read.answer(json(200, { total: 0, groups: [] })));
    await act(async () => button("common.options")?.click());
    expect(button("Week layout")?.querySelector("svg")).not.toBeNull();
    expect(button("Month layout")?.querySelector("svg")).toBeNull();
    await act(async () => button("common.actions.show_weekends")?.click());
    await until(() => writes().length === 1, "the weekends");
    const hidden = { ...page.display_filters, calendar: { layout: "week" as const, show_weekends: false } };
    expect(writes()[0]).toMatchObject({ path, body: { display_filters: hidden } });
    await act(async () => writes()[0]!.answer(json(200, { ...preferencesOf(), ...page, display_filters: hidden })));
    expect(days()).toEqual(["2026-12-28", "2026-12-29", "2026-12-30", "2026-12-31", "2027-01-01"]);
    await act(async () => button("Month layout")?.click());
    await until(() => writes().length === 2, "the month");
    const month = { ...hidden, calendar: { layout: "month" as const, show_weekends: false } };
    expect(writes()[1]).toMatchObject({ path, body: { display_filters: month } });
    await act(async () => writes()[1]!.answer(json(200, { ...preferencesOf(), ...page, display_filters: month })));
    await until(() => lists().length === 2, "the month's list");
    expect(lists()[1]!.query).toMatchObject({ target_date: "2026-11-29..2027-01-02", limit: "4" });
    expect(writes().every((call) => call.path === path)).toBe(true);
  }
);

it.each(["cycle", "project"] as const)(
  "the cycle page's calendar quick add follows its own flags, %s off",
  async (off) => {
    const flags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: true };
    if (off === "cycle") tab.app.issue.cycleIssues.viewFlags = flags;
    else tab.app.issue.projectIssues.viewFlags = flags;
    await load(pageOf(week(false)));
    const read = await mount(<BaseCalendarRoot QuickActions={NoActions} />);
    await act(async () => read.answer(json(200, dated)));
    expect(days()).toHaveLength(5);
    const adds = [...document.querySelectorAll("button")].filter((node) => node.textContent === "issue.add.label");
    expect(adds.length > 0).toBe(off === "project");
  }
);
