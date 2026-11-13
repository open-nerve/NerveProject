// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// A cycle's page's work items against a fake nerve (P5b spec 2.6), for the tests of its layout root: the route's
// project A with the caller of a role, its cycle C1 as nerve last gave it, and the page's preferences and its list as
// nerve answers them. The stores, rows, layouts and HTTP are real; the test file binds the stores and the route.
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { vi } from "vitest";
import type { Cycle, CyclePreferences, IssueListDisplayFilters, Project, ProjectIssueFilters } from "@nerve/api-client";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { listIssue } from "@/store/issue/fake-issues";
import { cycleOf, preferencesOf } from "@/store/project/fake-projects";
import { CycleLayoutRoot } from "./cycle-layout-root";

export const C1 = "019a0000-0000-7000-8000-0000000000c1";
export type Layout = "list" | "kanban" | "spreadsheet" | "calendar";

const first = listIssue("first", 1000, {
  cycle_id: C1,
  state_id: "state",
  priority: "high",
  target_date: "2026-12-31",
});
/** Each layout's display on the page, and nerve's list of the cycle for it, of its one work item or of none. */
export const layouts: Record<Layout, { display: Partial<IssueListDisplayFilters>; page: (rows?: boolean) => unknown }> =
  {
    list: {
      display: { layout: "list", group_by: "state_id", order_by: "sort_order" },
      page: (rows = true) => ({ total: rows ? 1 : 0, groups: rows ? [group("state")] : [] }),
    },
    kanban: {
      display: { layout: "kanban", group_by: "state_id", order_by: "sort_order" },
      page: (rows = true) => ({ total: rows ? 1 : 0, groups: rows ? [group("state")] : [] }),
    },
    spreadsheet: {
      display: { layout: "spreadsheet" },
      page: (rows = true) => ({ total: rows ? 1 : 0, data: rows ? [first] : [], next_cursor: null }),
    },
    calendar: {
      display: { layout: "calendar", calendar: { layout: "week", show_weekends: true } },
      page: (rows = true) => ({ total: rows ? 1 : 0, groups: rows ? [group("2026-12-31")] : [] }),
    },
  };
const group = (key: string) => ({ key, total: 1, data: [first], next_cursor: null });

/** The page's preferences as nerve answers them: the list defaults, with the page's own display and filters. */
export function cyclePreferencesOf(
  display: Partial<IssueListDisplayFilters> = {},
  filters: ProjectIssueFilters = {}
): CyclePreferences {
  const { display_filters, display_properties } = preferencesOf();
  return { filters, display_filters: { ...display_filters, ...display }, display_properties };
}

/**
 * A tab at the cycle's page: project A with the caller of role (a member's unless it says) and its fields, and its
 * cycle C1, current unless fields say otherwise. Today is Dec 31, 2026, the calendar's week Dec 27–Jan 2.
 */
export async function cyclePageTab(
  fields: { role?: Project["member_role"]; cycle?: Partial<Cycle>; project?: Partial<Project> } = {}
) {
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
  const cycle = cycleOf(tab.project, "Now", { id: C1, status: "current", ...fields.cycle });
  await answered(tab.nerve, () => tab.app.planning.cycles.fetchCycle(C1), ["GET", `/api/v0/cycles/${C1}`], cycle, "C1");
  tab.nerve.calls.length = 0;
  const preferences = () => tab.nerve.to(`/api/v0/me/cycles/${C1}/preferences`);
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  /** Renders the page's work items, after what else of the page the test puts before them (its header, its modals). */
  const mount = (before?: ReactNode) =>
    tab.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter>
          {before}
          <CycleLayoutRoot />
        </MemoryRouter>
      </SWRConfig>
    );
  return {
    ...tab,
    preferences,
    lists,
    mount,
    /** Renders the page's work items, as mount; nerve answers its preferences and, when given, its list. */
    show: async (page: CyclePreferences, list?: unknown, before?: ReactNode) => {
      await mount(before);
      await until(() => preferences().length === 1, "the page's preferences");
      await act(async () => preferences()[0]!.answer(json(200, page)));
      await until(() => lists().length === 1, "the page's list");
      if (list !== undefined) await act(async () => lists()[0]!.answer(json(200, list)));
    },
  };
}

/** The button of the page whose text is text. */
export const buttonOf = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);
