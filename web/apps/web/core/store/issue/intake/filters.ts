/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IntakeStatus, IssuePriority } from "@nerve/api-client";
// services
import type { IntakeListQuery } from "@/services/issue/intake.service";

/**
 * A range of days, each a day string (YYYY-MM-DD) of the caller's local calendar (R10): from the first to the last,
 * both in it; an end left out is open. nerve reads each day as a UTC day (C3).
 */
type DayRange = { from?: string; to?: string };

/**
 * The filters of a project's intake, as the intake page holds them (P5a spec 2.2): a tab, open or closed; the statuses
 * it lists, one at least (C1, R9); the priorities, assignees (ids, me, none), creators (ids, me) and labels (ids, none)
 * it lists, none for every one; the days of creation and of the last change, one range each (C3); the order, one of
 * nerve's six (C4).
 */
export type IntakeFilters = {
  tab: "open" | "closed";
  statuses: readonly [IntakeStatus, ...IntakeStatus[]];
  priorities: readonly IssuePriority[];
  assignees: readonly string[];
  creators: readonly string[];
  labels: readonly string[];
  created?: DayRange;
  updated?: DayRange;
  order: NonNullable<IntakeListQuery["order_by"]>;
};

/** A page of a project's intake holds 50 work items: nerve's default, sent all the same (C5, R6). */
const PAGE_SIZE = 50;

/** The values, each once, in order. */
const sortedOnce = <T extends string>(values: readonly T[]): T[] => [...new Set(values)].toSorted();

/** The ends the range has; none for a range of no end, which lists every day as no range does. */
const rangeOf = (range: DayRange | undefined): DayRange | undefined =>
  range === undefined || (range.from === undefined && range.to === undefined)
    ? undefined
    : { from: range.from, to: range.to };

/**
 * The filters in the one form of all the filters that list the same work items: each filter's values sorted, each
 * once, and a range of no end left out. filterKey names it and listQuery asks nerve for it, so that a list's key names
 * exactly what it fetches (P4 re-93 m5).
 */
const named = (filters: IntakeFilters) => ({
  tab: filters.tab,
  statuses: sortedOnce(filters.statuses),
  priorities: sortedOnce(filters.priorities),
  assignees: sortedOnce(filters.assignees),
  creators: sortedOnce(filters.creators),
  labels: sortedOnce(filters.labels),
  created: rangeOf(filters.created),
  updated: rangeOf(filters.updated),
  order: filters.order,
});

/**
 * @description the filters as one string, the same for any filters that list the same work items and different for
 * any others: the intake store keeps a project's lists by it, and the intake page's read of a list is keyed by it (P5a
 * spec 2.3, 2.5)
 */
export const filterKey = (filters: IntakeFilters): string => JSON.stringify(named(filters));

/**
 * A range of days as nerve takes it (C3): one day when both ends are the same, a..b, a.. or ..b; nothing for no range.
 * Never none: a range names days.
 */
function daysOf(range: DayRange | undefined): string | undefined {
  if (range === undefined) return undefined;
  if (range.from === range.to) return range.from;
  return `${range.from ?? ""}..${range.to ?? ""}`;
}

/**
 * @description the query of a page of the project's intake the filters select (P5a spec 2.2): always the statuses, in
 * either tab, the first ones too (C1: left out, nerve lists every status); the other filters only when they name some
 * values; each filter's values once, in order, as one array the client sends as one CSV parameter (C2: nerve refuses an
 * empty or a repeated parameter); each date as one range (C3); the order as it is, one of nerve's six (C4); pages of
 * 50 (C5); the cursor only as nerve answered it (C6). Never the tab: nerve takes none, the statuses select
 */
export function listQuery(filters: IntakeFilters, cursor?: string): IntakeListQuery {
  const { statuses, priorities, assignees, creators, labels, created, updated, order } = named(filters);
  const createdAt = daysOf(created);
  const updatedAt = daysOf(updated);
  return {
    status: statuses,
    ...(priorities.length > 0 ? { priority: priorities } : {}),
    ...(assignees.length > 0 ? { assignee_id: assignees } : {}),
    ...(creators.length > 0 ? { created_by_id: creators } : {}),
    ...(labels.length > 0 ? { label_id: labels } : {}),
    ...(createdAt === undefined ? {} : { created_at: createdAt }),
    ...(updatedAt === undefined ? {} : { updated_at: updatedAt }),
    order_by: order,
    limit: PAGE_SIZE,
    ...(cursor === undefined ? {} : { cursor }),
  };
}
