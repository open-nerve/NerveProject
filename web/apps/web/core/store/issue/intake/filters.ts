/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { format, subDays } from "date-fns";
// nerve imports
import type { IntakeIssue, IntakeStatus, IssuePriority } from "@nerve/api-client";
import type { INTAKE_DATE_PRESETS } from "@nerve/constants";
import { toggledOptions } from "@nerve/utils";
// services
import type { IntakeListQuery } from "@/services/issue/intake.service";
// local
import { effectiveStatus } from "./status";

/**
 * A range of days, each a day string (YYYY-MM-DD) of the caller's local calendar (R10): from the first to the last,
 * both in it; an end left out is open. nerve reads each day as a UTC day (C3).
 */
export type DayRange = { from?: string; to?: string };

/** A tab of the intake page: the work items still to triage (open), or those triaged (closed). */
export type IntakeTab = "open" | "closed";

/**
 * The filters of a project's intake, as the intake page holds them (P5a spec 2.2): a tab, open or closed; the statuses
 * it lists, one at least (C1, R9); the priorities, assignees (ids, me, none), creators (ids, me) and labels (ids, none)
 * it lists, none for every one; the days of creation and of the last change, one range each (C3); the order, one of
 * nerve's six (C4).
 */
export type IntakeFilters = {
  tab: IntakeTab;
  statuses: readonly [IntakeStatus, ...IntakeStatus[]];
  priorities: readonly IssuePriority[];
  assignees: readonly string[];
  creators: readonly string[];
  labels: readonly string[];
  created?: DayRange;
  updated?: DayRange;
  order: NonNullable<IntakeListQuery["order_by"]>;
};

/**
 * @description the filters a tab of the intake page opens with (P5a spec 2.2, R9): the open tab lists the pending work
 * items, the closed tab those accepted, declined or marked duplicates (C1: always one status at least); no other
 * filter; the newest first
 */
export const defaultFilters = (tab: IntakeTab): IntakeFilters => ({
  tab,
  statuses: tab === "open" ? ["pending"] : ["accepted", "declined", "duplicate"],
  priorities: [],
  assignees: [],
  creators: [],
  labels: [],
  order: "-created_at",
});

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
 * @description the filters a key of filterKey names, in their one form (named): what a list's read fetches is what
 * its key names, never the filters of the render that keyed it (P5a spec 2.5; P4 re-93 m5)
 */
export const filtersOf = (key: string): IntakeFilters => JSON.parse(key) as IntakeFilters;

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

/**
 * Whether a filter of ids selects a work item of the ids given, as nerve takes it: any of the filter's values does
 * (OR), me the caller (none when he is not known), none a work item of no id; a filter of no value selects every one.
 * The creators' filter never names none, nor the labels' me: nerve refuses either.
 */
function selects(filter: readonly string[], ids: readonly string[], me: string | undefined): boolean {
  return (
    filter.length === 0 ||
    filter.some((value) => {
      if (value === "none") return ids.length === 0;
      const id = value === "me" ? me : value;
      return id !== undefined && ids.includes(id);
    })
  );
}

/**
 * Whether the moment falls on a day of the range, as nerve takes it (C3): its UTC day, from the first day's midnight
 * UTC, in it, to the midnight after the last, out of it; an end left out is open, and no range takes every moment.
 */
function within(moment: string, range: DayRange | undefined): boolean {
  const day = new Date(moment).toISOString().slice(0, 10);
  return (range?.from === undefined || range.from <= day) && (range?.to === undefined || day <= range.to);
}

/**
 * @description whether a list of the filters lists the work item at now, for the caller me names, as nerve would
 * (P5a spec 2.2): its status at now among the statuses (effectiveStatus, C7); its priority, its assignees, its creator
 * and its labels among those each filter names (selects); the UTC days of its creation and of its last change in the
 * ranges (within, C3). The filters AND. Never the tab, which the statuses select, nor the order
 */
export function matches(row: IntakeIssue, filters: IntakeFilters, now: Date, me: string | undefined): boolean {
  return (
    filters.statuses.includes(effectiveStatus(row.intake, now)) &&
    (filters.priorities.length === 0 || filters.priorities.includes(row.priority)) &&
    selects(filters.assignees, row.assignee_ids, me) &&
    selects(filters.creators, row.created_by_id === null ? [] : [row.created_by_id], me) &&
    selects(filters.labels, row.label_ids, me) &&
    within(row.created_at, filters.created) &&
    within(row.updated_at, filters.updated)
  );
}

/**
 * @description the statuses a tab of the intake page can list (C1): the open tab those still to triage, pending and
 * snoozed; the closed tab those triaged, accepted, declined and marked duplicates
 */
export const tabStatuses = (tab: IntakeTab): readonly IntakeStatus[] =>
  tab === "open" ? ["pending", "snoozed"] : ["accepted", "declined", "duplicate"];

/** Whether the values name one at least. */
const isNonEmpty = <T>(values: T[]): values is [T, ...T[]] => values.length > 0;

/**
 * @description the statuses with status turned on, after those held, or off if held: the one rule of the filters'
 * menu and of the filters applied (R9). None when status is the last one on, which stays: a list names a status always
 * (C1: none named, nerve lists every status)
 */
export function toggledStatuses(
  statuses: IntakeFilters["statuses"],
  status: IntakeStatus
): IntakeFilters["statuses"] | undefined {
  const next = toggledOptions(statuses, [status]);
  return isNonEmpty(next) ? next : undefined;
}

/**
 * @description whether the filters list other work items than the tab's first ones (defaultFilters): a filter of the
 * user's own, the order aside, as an order lists the same work items. The filters applied show then, and a list of no
 * work item says that none matches them rather than the tab's own empty state
 */
export function customized(filters: IntakeFilters): boolean {
  const first = defaultFilters(filters.tab);
  return filterKey({ ...filters, order: first.order }) !== filterKey(first);
}

/** A preset of the days a list is filtered by (INTAKE_DATE_PRESETS). */
type DatePreset = (typeof INTAKE_DATE_PRESETS)[number]["value"];

/**
 * Each preset's first and last day, as days before today: today, yesterday, and the last 7 or 30 days from that many
 * days before today to today, as the intake's presets were before (getCustomDates).
 */
const PRESET_DAYS: Readonly<Record<DatePreset, readonly [number, number]>> = {
  today: [0, 0],
  yesterday: [1, 1],
  last_7_days: [7, 0],
  last_30_days: [30, 0],
};

/** The day of the caller's calendar so many days before now's, as a day string (YYYY-MM-DD). */
const localDay = (now: Date, daysBefore: number): string => format(subDays(now, daysBefore), "yyyy-MM-dd");

/**
 * @description the days a preset names at now, each a day of the caller's calendar, the first and the last in (R10):
 * nerve reads each as a UTC day (P2b R8, registered). Today and yesterday are one day each (a), the last 7 and 30 days
 * a range of them (a..b, C3)
 */
export function presetRange(preset: DatePreset, now: Date): DayRange {
  const [first, last] = PRESET_DAYS[preset];
  return { from: localDay(now, first), to: localDay(now, last) };
}

/**
 * @description the preset whose days the range is at now, if any: the menu marks it chosen, and the filters applied
 * name the range by it. A preset chosen on another day keeps the days it named then: now another preset's, or none
 */
export function presetOf(range: DayRange | undefined, now: Date): DatePreset | undefined {
  if (range === undefined) return undefined;
  return (Object.keys(PRESET_DAYS) as DatePreset[]).find((preset) => {
    const days = presetRange(preset, now);
    return days.from === range.from && days.to === range.to;
  });
}

/**
 * @description the days the custom dates' dialog chose (DateFilterModal's "<day>;after", "<day>;before", or both,
 * days of the caller's calendar): from the day after which, to the day before which, each in, as the dialog's other
 * readers take them; nerve's a.., ..b and a..b (C3), one range
 */
export function customRange(values: readonly string[]): DayRange {
  const range: DayRange = {};
  for (const value of values) {
    const [day, side] = value.split(";");
    if (side === "after") range.from = day;
    if (side === "before") range.to = day;
  }
  return range;
}
