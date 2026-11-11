// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A layout can be read-only as a whole (P5b spec 2.6, R5, re-66 M2): an ended or archived cycle's page passes it to
// the four layouts, which then offer no quick add, group "+", inline edit or drag, and their quick actions are
// read-only, although the caller may write in the project. Without it each layout offers them as the project page does.
import { act, createElement, type FC, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { CyclePreferences, IssueListDisplayFilters } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { listIssue } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import { BaseCalendarRoot } from "./calendar/base-calendar-root";
import { BaseKanBanRoot } from "./kanban/base-kanban-root";
import { BaseListRoot } from "./list/base-list-root";
import type { IQuickActionProps } from "./list/list-view-types";
import { BaseSpreadsheetRoot } from "./spreadsheet/base-spreadsheet-root";

type Drop = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements>[0];
type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  drops: new Map<Element, Drop>(),
  drags: new Map<Element, Drag>(),
}));
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
// jsdom has no viewport geometry or drag transport; rows, cards, days, dropdowns and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  dropTargetForElements: (options: Drop) => {
    bound.drops.set(options.element, options);
    return () => bound.drops.delete(options.element);
  },
  draggable: (options: Drag) => {
    bound.drags.set(options.element, options);
    return () => bound.drags.delete(options.element);
  },
}));

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const first = listIssue("first", 1000, {
  cycle_id: C1,
  state_id: "state",
  priority: "high",
  target_date: "2026-12-31",
});
/** The row's quick actions say whether they were given read-only. */
const Recorded = ({ issue, readOnly }: IQuickActionProps) => (
  <output data-quick-actions={issue.id}>{readOnly ? "read-only" : "writable"}</output>
);
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const writes = () => tab.nerve.calls.filter((call) => call.method !== "GET");
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  bound.drops.clear();
  bound.drags.clear();
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
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
});
afterEach(async () => tab.close());

type Layout = "list" | "kanban" | "spreadsheet" | "calendar";
type Root = FC<{ QuickActions: FC<IQuickActionProps>; readOnly?: boolean }>;
/** Each layout's root, its page display, the list nerve answers, and which writes it offers when it may write. */
const layouts: Record<
  Layout,
  {
    Root: Root;
    display: Partial<IssueListDisplayFilters>;
    page: unknown;
    offers: { groupCreate: boolean; inlineEdit: boolean; drag: boolean };
  }
> = {
  list: {
    Root: BaseListRoot,
    display: { layout: "list", group_by: "state_id", order_by: "sort_order" },
    page: { total: 1, groups: [{ key: "state", total: 1, data: [first], next_cursor: null }] },
    offers: { groupCreate: true, inlineEdit: true, drag: true },
  },
  kanban: {
    Root: BaseKanBanRoot,
    display: { layout: "kanban", group_by: "state_id", order_by: "sort_order" },
    page: { total: 1, groups: [{ key: "state", total: 1, data: [first], next_cursor: null }] },
    offers: { groupCreate: true, inlineEdit: true, drag: true },
  },
  spreadsheet: {
    Root: BaseSpreadsheetRoot,
    display: { layout: "spreadsheet" },
    page: { total: 1, data: [first], next_cursor: null },
    offers: { groupCreate: false, inlineEdit: true, drag: false },
  },
  calendar: {
    Root: BaseCalendarRoot,
    display: { layout: "calendar", calendar: { layout: "week", show_weekends: true } },
    page: { total: 1, groups: [{ key: "2026-12-31", total: 1, data: [first], next_cursor: null }] },
    offers: { groupCreate: false, inlineEdit: false, drag: true },
  },
};

/** Shows the layout on the cycle's page with the page's display and one row. */
async function show(layout: Layout, readOnly?: boolean) {
  const { Root, display, page } = layouts[layout];
  const { filters, display_filters, display_properties } = preferencesOf();
  const preferences: CyclePreferences = {
    filters,
    display_filters: { ...display_filters, ...display },
    display_properties,
  };
  await answered(
    tab.nerve,
    () => tab.app.issue.cycleIssuesFilter.fetchFilters("ws", C1),
    ["GET", `/api/v0/me/cycles/${C1}/preferences`],
    preferences,
    "the cycle page's preferences"
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <Root QuickActions={Recorded} readOnly={readOnly} />
      </MemoryRouter>
    </SWRConfig>
  );
  await until(() => lists().length === 1, "the page's list");
  await act(async () => lists()[0]!.answer(json(200, page)));
  expect(document.querySelectorAll('[id^="issue-first"]').length).toBeGreaterThan(0);
}
/** What the shown layout offers: creation, inline edits of the row, dragging it, and its quick actions' mode. */
function offered() {
  const buttons = [...document.querySelectorAll<HTMLButtonElement>("button")];
  return {
    quickAdd: buttons.some((node) => ["issue.new", "issue.add.label"].includes(node.textContent?.trim() ?? "")),
    groupCreate: !!document.querySelector('button[aria-label^="create_work_item: "]'),
    inlineEdit: priorities().some((node) => !node.disabled),
    drag: [...bound.drags.values()].some((drag) => drag.canDrag?.({} as never) ?? true),
    quickActions: [...new Set([...document.querySelectorAll("[data-quick-actions]")].map((node) => node.textContent))],
  };
}
const priorities = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((node) =>
    node.textContent?.startsWith("common.priority: ")
  );

it.each(Object.keys(layouts) as Layout[])(
  "a read-only %s offers no quick add, group create, inline edit or drag, and its quick actions are read-only",
  async (layout) => {
    await show(layout, true);
    const { offers } = layouts[layout];
    if (offers.inlineEdit) expect(priorities().length, "the row's priority").toBeGreaterThan(0);
    if (offers.drag) expect(bound.drags.size, "the row's drag source").toBeGreaterThan(0);
    expect(offered()).toEqual({
      quickAdd: false,
      groupCreate: false,
      inlineEdit: false,
      drag: false,
      quickActions: ["read-only"],
    });
    expect(writes()).toEqual([]);
  }
);

it.each(Object.keys(layouts) as Layout[])(
  "without read-only the %s offers each of them to a member, as the project page does",
  async (layout) => {
    await show(layout);
    expect(offered()).toEqual({ quickAdd: true, ...layouts[layout].offers, quickActions: ["writable"] });
    expect(priorities().every((node) => !node.disabled)).toBe(true);
  }
);

it.each([true, false])(
  "a drop on the calendar's day changes the row only when not read-only (%s)",
  async (readOnly) => {
    await show("calendar", readOnly);
    const day = [...bound.drops.values()].find((drop) => drop.getData?.({} as never)?.date === "2026-12-30");
    expect(day, "the day before the row's").toBeDefined();
    await act(async () =>
      day!.onDrop!({
        source: { data: { id: "first", date: "2026-12-31" } },
        self: { data: { date: "2026-12-30" } },
      } as never)
    );
    if (readOnly) {
      await act(async () => vi.advanceTimersByTimeAsync(1000));
      return expect(writes()).toEqual([]);
    }
    await until(() => writes().length === 1, "the drop");
    expect(writes()[0]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/issues/first",
      body: { target_date: "2026-12-30" },
    });
  }
);
