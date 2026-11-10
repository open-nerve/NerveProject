/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { sortBy } from "lodash-es";
// nerve imports
import type { Cycle, CycleStatus, FrozenCounts } from "@nerve/api-client";
// local imports
import { getDate } from "./datetime";
import { satisfiesDateFilter } from "./filter";

// The planning of a project as its pages show it (P5a spec 2.3): the order of its cycles list, its filters, and the
// one reading of a cycle's progress. A cycle's status is nerve's, never worked out here from its days; its days are
// days of the project's calendar ("YYYY-MM-DD", M6 design 3.1), read as the browser's local days, never as times.

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
