/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { sortBy } from "lodash-es";
// nerve imports
import type { Cycle, CycleStatus, FrozenCounts, Module, ModuleListItem, ModuleStatus } from "@nerve/api-client";
import type { ModuleOrderBy } from "@nerve/constants";
// local imports
import { getDate } from "./datetime";
import { satisfiesDateFilter } from "./filter";

// The planning of a project as its pages show it (P5a spec 2.3, P6a spec 2.4): the order of its cycles list and of
// its modules list, their filters, and the one reading of a cycle's (or a module's) progress. A status is nerve's,
// never worked out here from the days; the days are days of the project's calendar ("YYYY-MM-DD", M6 design 3.1),
// read as the browser's local days, never as times.

/** The six counts of work items a cycle's row, its progress and what either was frozen with each have. */
type WorkItemCounts = Omit<FrozenCounts, "frozen_at">;

/**
 * What a cycles list is filtered by (P5a spec 2.3): nerve's status, any of those chosen; and the days a cycle starts
 * and ends on, every option chosen, as the filter menus write them: "1_weeks;after;fromnow" (a week from today at the
 * earliest; also 2_weeks, 1_months, 2_months), or a custom day, "YYYY-MM-DD;after" or "YYYY-MM-DD;before" (the day
 * itself included). A filter with nothing chosen filters nothing.
 */
export type CycleFilters = {
  readonly status?: readonly CycleStatus[];
  readonly start_date?: readonly string[];
  readonly end_date?: readonly string[];
};

/** Where each status of the list's first parts goes; the ended cycles are a part of their own, in nerve's order (R5). */
const ORDER: Partial<Record<CycleStatus, number>> = { current: 1, upcoming: 2, draft: 3 };

/**
 * The cycles in progress, upcoming and drafts, in the list's order (P5a spec 2.3, R5): in progress first, then the
 * upcoming ones by the day they start, then the drafts by their names' lower case. The ended cycles are left out.
 */
export const orderCycleRows = <T extends Cycle>(rows: readonly T[]): T[] =>
  sortBy(
    rows.filter((row) => ORDER[row.status] !== undefined),
    [(row) => ORDER[row.status], (row) => (row.status === "upcoming" ? row.start_date : row.name.toLowerCase())]
  );

/** Whether a day of the project's calendar, if the cycle has it, satisfies every date option chosen. */
const matchesDays = (day: string | null, options: readonly string[] | undefined): boolean => {
  if (!options?.length) return true;
  // getDate reads "YYYY-MM-DD" as that day's local midnight; satisfiesDateFilter counts days from the local today.
  const date = getDate(day);
  return !!date && options.every((option) => satisfiesDateFilter(date, option));
};

/** Whether a cycle (or its row) passes a cycles list's filters (P5a spec 2.3). */
export const matchesCycleFilters = (cycle: Cycle, filters: CycleFilters): boolean =>
  (!filters.status?.length || filters.status.includes(cycle.status)) &&
  matchesDays(cycle.start_date, filters.start_date) &&
  matchesDays(cycle.end_date, filters.end_date);

/**
 * The counts a cycle shows (P5a spec 2.3, R7; M6 design 3.5): those it was frozen with once it has ended and been
 * frozen, else its live ones. A row of a list gives its counts and its progress_snapshot; a cycle's progress, its
 * live progress and its progress_snapshot, whose distributions and burndown go with the counts chosen. The one place
 * the condition is written: nerve keeps a snapshot of a cycle whose days changed so that it is no longer ended.
 */
export const cycleCounts = <L extends WorkItemCounts, F extends WorkItemCounts>(
  status: CycleStatus,
  live: L,
  frozen: F | null
): L | F => (status === "completed" && frozen !== null ? frozen : live);

/**
 * How far a cycle (or a module) has come, in percent (P5a spec 2.3; M1 design 3.4): its completed work items of
 * those not cancelled, rounded to the nearest whole percent; 0 without any not cancelled.
 */
export const progressPercent = (
  counts: Pick<WorkItemCounts, "total_issues" | "completed_issues" | "cancelled_issues">
): number => {
  const counted = counts.total_issues - counts.cancelled_issues;
  return counted > 0 ? Math.round((counts.completed_issues / counted) * 100) : 0;
};

/**
 * What a modules list is filtered by (P6a spec 2.4): nerve's status, any of those chosen; the lead, any of those
 * chosen; the members, a module with any of those chosen as a member; and the days it starts and is due on, every
 * option chosen, written as a cycles list's are (CycleFilters). A filter with nothing chosen filters nothing.
 */
export type ModuleFilters = {
  readonly status?: readonly ModuleStatus[];
  readonly lead?: readonly string[];
  readonly members?: readonly string[];
  readonly start_date?: readonly string[];
  readonly target_date?: readonly string[];
};

/** Whether a module (or its row) passes a modules list's filters (P6a spec 2.4). */
export const matchesModuleFilters = (module: Module, filters: ModuleFilters): boolean =>
  (!filters.status?.length || filters.status.includes(module.status)) &&
  (!filters.lead?.length || (module.lead_id !== null && filters.lead.includes(module.lead_id))) &&
  (!filters.members?.length || filters.members.some((member) => module.member_ids.includes(member))) &&
  matchesDays(module.start_date, filters.start_date) &&
  matchesDays(module.target_date, filters.target_date);

/** An option of the modules list's order without its direction. */
type ModuleOrderKey = Exclude<ModuleOrderBy, `-${string}`>;

/** Names in their natural order: the numbers within them as numbers ("Sprint 9" before "Sprint 10"), any case. */
const collator = new Intl.Collator("en-US", { numeric: true, sensitivity: "base" });

/**
 * What a row is ordered by, for each option: a name, or a number. A target day is that day's local midnight (getDate,
 * never new Date("YYYY-MM-DD")); a row without one has none. A moment nerve wrote is read as a time: nerve writes the
 * fraction of a second it has, so the texts are not in the moments' order.
 */
const ORDER_VALUE: Record<ModuleOrderKey, (row: ModuleListItem) => string | number | null> = {
  name: (row) => row.name,
  progress: progressPercent,
  issues_length: (row) => row.total_issues,
  target_date: (row) => getDate(row.target_date)?.getTime() ?? null,
  created_at: (row) => Date.parse(row.created_at),
  sort_order: (row) => row.sort_order,
};

const compareValues = (a: string | number, b: string | number): number =>
  typeof a === "number" && typeof b === "number" ? a - b : collator.compare(String(a), String(b));

/**
 * A project's modules in the order its list and archive show them (P6a spec 2.4, R14): by the option chosen, the
 * lowest first, or the highest first behind a "-"; a progress is progressPercent's, the one reading of it. Rows that
 * order the same keep the order they were given (nerve's); a row without a target day goes last, whichever the
 * direction. What was given is left as it was.
 */
export const orderModuleRows = <T extends ModuleListItem>(rows: readonly T[], orderBy: ModuleOrderBy): T[] => {
  const descending = orderBy.startsWith("-");
  const valueOf = ORDER_VALUE[(descending ? orderBy.slice(1) : orderBy) as ModuleOrderKey];
  return rows.toSorted((a, b) => {
    const [x, y] = [valueOf(a), valueOf(b)];
    if (x === null || y === null) return Number(x === null) - Number(y === null);
    return descending ? compareValues(y, x) : compareValues(x, y);
  });
};
