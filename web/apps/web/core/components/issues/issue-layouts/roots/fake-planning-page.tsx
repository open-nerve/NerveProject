// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A planning page's work items against a fake nerve (P5b spec 2.6, P6b spec 2.2), per domain, for the tests of the
// shared layout root: the route's project A with the caller of a role, the page's cycle C1 or module M1 as nerve last
// gave it, and the page's preferences and its list as nerve answers them. The stores, rows, layouts and HTTP are real;
// the test file binds the stores and the route. Below them, what the pages' tests do on the page the test opened: its
// header's buttons, a row's quick actions and dialogs, and a new work item's form (P6b review re-71 M1).
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { expect, vi } from "vitest";
import type {
  Cycle,
  CyclePreferences,
  Issue,
  IssueDetail,
  IssueListDisplayFilters,
  Module,
  ModuleIssueFilters,
  ModulePreferences,
  Project,
  ProjectIssueFilters,
} from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { PeekOnEntry } from "@/components/cycles/peek-on-entry";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { loadModule } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf, preferencesOf } from "@/store/project/fake-projects";
import { PlanningPageLayoutRoot } from "./planning-page-layout-root";

export const C1 = "019a0000-0000-7000-8000-0000000000c1";
export const M1 = "019a0000-0000-7000-8000-0000000000d1";
export type Layout = "list" | "kanban" | "spreadsheet" | "calendar";

const shown = { state_id: "state", priority: "high", target_date: "2026-12-31" } as const;
/** The page's one work item "first": the cycle's, or the module's. */
const first = listIssue("first", 1000, { cycle_id: C1, ...shown });
const moduleFirst = listIssue("first", 1000, { module_ids: [M1], ...shown });
type Page = (rows?: boolean, row?: Issue) => unknown;
/**
 * Each layout's display on the page, and nerve's list of the page for it, of its one work item (the cycle's unless
 * given) or of none.
 */
export const layouts: Record<Layout, { display: Partial<IssueListDisplayFilters>; page: Page }> = {
  list: {
    display: { layout: "list", group_by: "state_id", order_by: "sort_order" },
    page: (rows = true, row = first) => ({ total: rows ? 1 : 0, groups: rows ? [group("state", row)] : [] }),
  },
  kanban: {
    display: { layout: "kanban", group_by: "state_id", order_by: "sort_order" },
    page: (rows = true, row = first) => ({ total: rows ? 1 : 0, groups: rows ? [group("state", row)] : [] }),
  },
  spreadsheet: {
    display: { layout: "spreadsheet" },
    page: (rows = true, row = first) => ({ total: rows ? 1 : 0, data: rows ? [row] : [], next_cursor: null }),
  },
  calendar: {
    display: { layout: "calendar", calendar: { layout: "week", show_weekends: true } },
    page: (rows = true, row = first) => ({ total: rows ? 1 : 0, groups: rows ? [group("2026-12-31", row)] : [] }),
  },
};
const group = (key: string, row: Issue) => ({ key, total: 1, data: [row], next_cursor: null });

/** The page's preferences as nerve answers them: the list defaults, with the page's own display and filters. */
export function cyclePreferencesOf(
  display: Partial<IssueListDisplayFilters> = {},
  filters: ProjectIssueFilters = {}
): CyclePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters, display_filters: { ...display_filters, ...display }, display_properties };
}

/** The module page's preferences as nerve answers them: the list defaults, with the page's own display and filters. */
export function modulePreferencesOf(
  display: Partial<IssueListDisplayFilters> = {},
  filters: ModuleIssueFilters = {}
): ModulePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters, display_filters: { ...display_filters, ...display }, display_properties };
}

type Tab = Awaited<ReturnType<typeof issueSelectionTab>>;
/**
 * What a test says of the page: the caller's role (a member's unless it says), its project's fields, its cycle's or its
 * module's.
 */
type PageFields = {
  role?: Project["member_role"];
  project?: Partial<Project>;
  cycle?: Partial<Cycle>;
  module?: Partial<Module>;
};

/** What each domain's page is to its tests (the page's store type picks it). */
const pages: Record<
  PlanningPageStoreType,
  {
    /** Where nerve keeps the page's preferences. */
    preferences: string;
    /** Has the tab hold the page's cycle or module as nerve gives it, as the fields say. */
    hold: (tab: Tab, fields: PageFields) => Promise<void>;
    /** The page's one work item "first", as nerve lists it. */
    row: Issue;
    /** What the page puts beside its preview host (PlanningPageLayoutRoot's children). */
    beside: ReactNode;
  }
> = {
  [EIssuesStoreType.CYCLE]: {
    preferences: `/api/v0/me/cycles/${C1}/preferences`,
    hold: async (tab, fields) => {
      const cycle = cycleOf(tab.project, "Now", { id: C1, status: "current", ...fields.cycle });
      await answered(
        tab.nerve,
        () => tab.app.planning.cycles.fetchCycle(C1),
        ["GET", `/api/v0/cycles/${C1}`],
        cycle,
        "C1"
      );
    },
    row: first,
    // the cycle's page: the preview the way in asked for (PeekOnEntry)
    beside: <PeekOnEntry workspaceSlug="ws" projectId="A" />,
  },
  [EIssuesStoreType.MODULE]: {
    preferences: `/api/v0/me/modules/${M1}/preferences`,
    hold: async (tab, fields) => {
      await loadModule(
        tab.nerve,
        tab.app.planning.modules,
        moduleOf(tab.project, "Core", { id: M1, ...fields.module })
      );
    },
    row: moduleFirst,
    // a module's page has nothing beside its preview host (no way in asks for a preview)
    beside: null,
  },
};

/** The page's work items as the page of the store type puts them: the shared layout root, with what is beside it. */
export function PlanningPageWorkItems(props: { type: PlanningPageStoreType }) {
  return <PlanningPageLayoutRoot storeType={props.type}>{pages[props.type].beside}</PlanningPageLayoutRoot>;
}

/**
 * A tab at the store type's page: project A with the caller of role (a member's unless it says) and its fields, and
 * the page's cycle C1, current, or its module M1, in its backlog, unless fields say otherwise. Today is Dec 31, 2026,
 * the calendar's week Dec 27–Jan 2.
 */
export async function planningPageTab(type: PlanningPageStoreType, fields: PageFields = {}) {
  const tab = await issueSelectionTab();
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
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({
    ...project,
    member_role: fields.role ?? 15,
    ...fields.project,
  }));
  await tab.app.state.fetchProjectStates("A");
  await pages[type].hold(tab, fields);
  tab.nerve.calls.length = 0;
  const preferences = () => tab.nerve.to(pages[type].preferences);
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  /** Renders the page's work items, after what else of the page the test puts before them (its header, its modals). */
  const mount = (before?: ReactNode) =>
    tab.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter>
          {before}
          <PlanningPageWorkItems type={type} />
        </MemoryRouter>
      </SWRConfig>
    );
  const planningTab = {
    ...tab,
    preferences,
    lists,
    mount,
    /** nerve's list of the page for layout, of the page's one work item or of none. */
    listOf: (layout: Layout, rows = true) => layouts[layout].page(rows, pages[type].row),
    /** Renders the page's work items, as mount; nerve answers its preferences and, when given, its list. */
    show: async (page: CyclePreferences | ModulePreferences, list?: unknown, before?: ReactNode) => {
      await mount(before);
      await until(() => preferences().length === 1, "the page's preferences");
      await act(async () => preferences()[0]!.answer(json(200, page)));
      await until(() => lists().length === 1, "the page's list");
      if (list !== undefined) await act(async () => lists()[0]!.answer(json(200, list)));
    },
  };
  opened = planningTab;
  return planningTab;
}
type PlanningPageTab = Awaited<ReturnType<typeof planningPageTab>>;
/** The tab planningPageTab opened last: the test's page, which the helpers below act on (a test has one at a time). */
let opened: PlanningPageTab | undefined;
function current(): PlanningPageTab {
  if (!opened) throw new Error("no planning page is open: planningPageTab opens one");
  return opened;
}

/** The button of the page whose text is text. */
export const buttonOf = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);

/** The header's button of name: its aria-label, or else its text. */
export const inHeader = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("[data-header] button")].find(
    (node) => (node.getAttribute("aria-label") ?? node.textContent?.trim()) === name
  );
/** The header's button of text. */
export const headerButton = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("[data-header] button")].find(
    (node) => node.textContent?.trim() === text
  );

/** Whether the row is on the page: its quick actions are its own. */
export const rowShown = () => !!document.querySelector('[data-context-menu="true"] [role="menuitem"]');
/** The title the open dialog shows, null without a dialog. */
export const dialogTitle = () =>
  document.querySelector<HTMLInputElement>('[role="dialog"] input[name="name"]')?.value ?? null;
/** Chooses the row's quick action of name. */
export const choose = (name: string) =>
  act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>('[data-context-menu="true"] [role="menuitem"]')]
      .find((node) => node.textContent === name)
      ?.click()
  );
/** The page's work item "first" as nerve reads it: the row's own fields, with its description. */
const detail = (): IssueDetail => {
  const [row] = (current().listOf("spreadsheet") as { data: IssueDetail[] }).data;
  return { ...issueDetailOf("first"), ...row };
};
/** The reads of the work item itself: the dialog's, then the check its sub-issues' read makes. */
const reads = () =>
  current()
    .nerve.to("/api/v0/issues/first")
    .filter((call) => call.method === "GET");

/**
 * A row's edit and copy dialogs outlive the row (P5b T13): chooses the row's quick action of name and answers the
 * dialog's reads one at a time: the dialog shows title from the start, and still while a read is out with the row off
 * the page (the check of the work item); the list's read again brings the row back, and the dialog still shows title.
 */
export async function outlivesTheRow(name: string, title: string, list: unknown) {
  const page = current();
  await until(rowShown, "the row");
  const normal = page.nerve.respond;
  page.nerve.respond = (call: Call) => {
    if (
      call.method === "GET" &&
      /\/issues\/first\/(sub-issues|relations|links|reactions|activities|comments)$/.test(call.path)
    )
      call.answer(json(200, { data: [], next_cursor: null }));
    else normal?.(call);
  };
  await choose(name);
  expect(dialogTitle()).toBe(title);
  const answeredReads = new Set<Call>();
  const out = () => reads().find((call) => !answeredReads.has(call));
  const titlesWhileOff: (string | null)[] = [];
  // the dialog's read, then the check of the work item, each answered as nerve reads it, until the list reads again
  const answerReads = async (left: number): Promise<void> => {
    await until(() => out() !== undefined || page.lists().length === 2, "a read of the work item, or of the list");
    const read = out();
    if (read === undefined) return;
    if (left === 0) throw new Error("the work item was read more than five times");
    if (!rowShown()) titlesWhileOff.push(dialogTitle());
    answeredReads.add(read);
    await act(async () => read.answer(json(200, detail())));
    return answerReads(left - 1);
  };
  await answerReads(5);
  expect(titlesWhileOff).toEqual([title]);
  await act(async () => page.lists()[1]!.answer(json(200, list)));
  await until(rowShown, "the row again");
  expect(dialogTitle()).toBe(title);
}

/** The page's creations: what was sent to nerve to create a work item in project A. */
export const creates = () =>
  current()
    .nerve.to("/api/v0/projects/A/issues")
    .filter((call) => call.method === "POST");
/** The open dialog's form. */
export const form = () => document.querySelector<HTMLFormElement>('[role="dialog"] form');
/** The form's project field: its dropdown's button, which names the project. */
export const projectField = () =>
  [...(form()?.querySelectorAll<HTMLButtonElement>("button") ?? [])].find(
    (node) => node.textContent === "common.project: A"
  );
/** Types the name of the new work item in the input of selector and sends what it is in. */
export async function send(selector: string, name: string, at: () => HTMLFormElement | null) {
  await act(async () => {
    const input = at()!.querySelector<HTMLInputElement>(selector)!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, name);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => at()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => creates().length === 1, "the creation");
}
/** The quick add's form, in the list, not in a dialog. */
export const quickAddForm = () =>
  [...document.querySelectorAll<HTMLFormElement>("form")].find((node) => !node.closest('[role="dialog"]')) ?? null;
/** nerve's answer to a creation of name: the work item "made", with fields. */
export const made = (name: string, fields: object = {}) => ({
  ...issueDetailOf("made"),
  name,
  sort_order: 500,
  ...fields,
});
/** Sends a new work item from the app's own new work item, of no page, and nerve's answer to it. */
export async function globalCreate(name: string) {
  await act(async () => current().app.commandPalette.toggleCreateIssueModal(true));
  expect(projectField()?.disabled, "the app's form lets the project be changed").toBe(false);
  await send("#name", name, form);
}
