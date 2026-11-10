/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  Cycle,
  CycleListItem,
  CycleProgress,
  CycleStatus,
  FrozenCounts,
  FrozenProgress,
  Progress,
} from "@nerve/api-client";
import { cycleCounts, matchesCycleFilters, orderCycleRows, progressPercent } from "./planning";

// The planning of a project as its pages show it (P5a spec 2.3, R5, R7): the order of its cycles list, its filters,
// and the one reading of a cycle's progress. A cycle's status is nerve's, never worked out here from its dates; its
// dates are days of the project's calendar ("YYYY-MM-DD"), compared with the browser's today as days.

/** A cycle of web, of nerve's status, and of the days it starts and ends on, if any. */
const cycleOf = (
  name: string,
  status: CycleStatus,
  start_date: string | null = null,
  end_date = start_date
): Cycle => ({
  id: `c-${name}`,
  workspace_id: "w-acme",
  project_id: "p-web",
  name,
  description: "",
  start_date,
  end_date,
  status,
  owned_by_id: null,
  sort_order: 65535,
  archived_at: null,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
  created_by_id: null,
  updated_by_id: null,
});
/** The six counts of a cycle's work items: its total, of which some completed and some cancelled; the rest started. */
const countsOf = (total_issues: number, completed_issues: number, cancelled_issues = 0) => ({
  total_issues,
  backlog_issues: 0,
  unstarted_issues: 0,
  started_issues: total_issues - completed_issues - cancelled_issues,
  completed_issues,
  cancelled_issues,
});
/** A cycle's row in its list, with its live counts, and those it was frozen with, if it was. */
const rowOf = (cycle: Cycle, live = countsOf(0, 0), progress_snapshot: FrozenCounts | null = null): CycleListItem => ({
  ...cycle,
  ...live,
  assignee_ids: [],
  progress_snapshot,
});
const frozenOf = (total: number, completed: number): FrozenCounts => ({
  ...countsOf(total, completed),
  frozen_at: "2026-11-02T00:00:00Z",
});
const names = (cycles: Pick<Cycle, "name">[]) => cycles.map((cycle) => cycle.name);

describe("orderCycleRows", () => {
  it("orders the cycles in progress, upcoming and drafts as the list shows them, and leaves the ended ones out", () => {
    // nerve's order, by sort order; the ended cycles keep it in their own part of the page (R5).
    const rows = [
      rowOf(cycleOf("zeta", "draft")),
      rowOf(cycleOf("S4", "upcoming", "2026-11-23")),
      rowOf(cycleOf("S1", "completed", "2026-10-26")),
      rowOf(cycleOf("Ideas", "draft")),
      rowOf(cycleOf("S2", "current", "2026-11-09")),
      rowOf(cycleOf("Alpha", "draft")),
      rowOf(cycleOf("A3", "upcoming", "2026-11-16")),
    ];
    // In progress first; then the upcoming ones by the day they start (not by name); then the drafts by their names'
    // lower case ("zeta" after "Ideas", where the names themselves would put "Ideas" after "Alpha" and before "zeta").
    expect(names(orderCycleRows(rows))).toEqual(["S2", "A3", "S4", "Alpha", "Ideas", "zeta"]);
  });

  it("orders by nerve's status alone, whatever the cycles' days say", () => {
    // nerve last said S5, which starts before S6, is upcoming, and S6 in progress: the page shows nerve's word.
    const rows = [
      rowOf(cycleOf("S5", "upcoming", "2026-11-01", "2026-11-30")),
      rowOf(cycleOf("S6", "current", "2026-12-01")),
    ];
    expect(names(orderCycleRows(rows))).toEqual(["S6", "S5"]);
  });
});

describe("matchesCycleFilters", () => {
  // A zone west of UTC, where a "YYYY-MM-DD" read as a time (UTC's midnight) falls on the day before, and today's
  // evening is already tomorrow in UTC: a date read through new Date("YYYY-MM-DD"), or today taken from UTC, is a day
  // off here. The zone is the test's alone (vi.stubEnv); the CI's zone is UTC, where neither shows.
  beforeAll(() => {
    vi.stubEnv("TZ", "America/Los_Angeles");
  });
  afterAll(() => {
    vi.unstubAllEnvs();
  });
  // The browser's today: Wednesday 11 November 2026, in the evening (in UTC, the 12th already).
  beforeEach(() => {
    vi.useFakeTimers({ now: new Date(2026, 10, 11, 21, 30) });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("runs in a zone west of UTC", () => {
    expect(new Date(2026, 10, 11).getTimezoneOffset()).toBe(480);
    expect(new Date().toISOString().slice(0, 10)).toBe("2026-11-12");
  });

  it("filters by nerve's status, any of those chosen; none chosen filters nothing", () => {
    const filters = { status: ["upcoming", "draft"] as CycleStatus[] };
    // S5's days have come, but nerve says it is upcoming: it is.
    expect(matchesCycleFilters(cycleOf("S5", "upcoming", "2026-11-01", "2026-11-30"), filters)).toBe(true);
    expect(matchesCycleFilters(cycleOf("Ideas", "draft"), filters)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S2", "current", "2026-11-09", "2026-11-15"), filters)).toBe(false);
    expect(matchesCycleFilters(cycleOf("S2", "current", "2026-11-09", "2026-11-15"), { status: [] })).toBe(true);
    expect(matchesCycleFilters(cycleOf("S2", "current", "2026-11-09", "2026-11-15"), {})).toBe(true);
  });

  it.each([
    { option: "1_weeks;after;fromnow", first: "2026-11-18", dayBefore: "2026-11-17" },
    { option: "2_weeks;after;fromnow", first: "2026-11-25", dayBefore: "2026-11-24" },
    { option: "1_months;after;fromnow", first: "2026-12-11", dayBefore: "2026-12-10" },
    { option: "2_months;after;fromnow", first: "2027-01-10", dayBefore: "2027-01-09" },
  ])("filters $option: a start on $first at the earliest, counting days from today", ({ option, first, dayBefore }) => {
    const filters = { start_date: [option] };
    expect(matchesCycleFilters(cycleOf("S", "upcoming", first, "2027-03-01"), filters)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S", "upcoming", dayBefore, "2027-03-01"), filters)).toBe(false);
  });

  it("filters by a custom day, the day itself included, after or before it", () => {
    const after = { start_date: ["2026-11-18;after"] };
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-18", "2026-11-24"), after)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-17", "2026-11-24"), after)).toBe(false);
    const before = { end_date: ["2026-11-18;before"] };
    expect(matchesCycleFilters(cycleOf("S", "current", "2026-11-09", "2026-11-18"), before)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-12", "2026-11-19"), before)).toBe(false);
  });

  it("filters by every date option chosen, and by every kind of filter at once", () => {
    // A custom range: an end on the 16th to the 20th.
    const range = { end_date: ["2026-11-16;after", "2026-11-20;before"] };
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-14", "2026-11-20"), range)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S", "current", "2026-11-09", "2026-11-15"), range)).toBe(false);
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-16", "2026-11-21"), range)).toBe(false);
    const both = { status: ["upcoming"] as CycleStatus[], start_date: ["1_weeks;after;fromnow"] };
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-23", "2026-11-29"), both)).toBe(true);
    expect(matchesCycleFilters(cycleOf("S", "upcoming", "2026-11-12", "2026-11-18"), both)).toBe(false);
    expect(matchesCycleFilters(cycleOf("S", "draft"), both)).toBe(false);
  });

  it("leaves out a cycle without days when its dates are filtered, and keeps it when they are not", () => {
    expect(matchesCycleFilters(cycleOf("Ideas", "draft"), { start_date: ["2026-11-01;after"] })).toBe(false);
    expect(matchesCycleFilters(cycleOf("Ideas", "draft"), { end_date: ["2027-01-01;before"] })).toBe(false);
    expect(matchesCycleFilters(cycleOf("Ideas", "draft"), { start_date: [], end_date: [] })).toBe(true);
  });
});

describe("cycleCounts", () => {
  const live = countsOf(5, 1);
  const frozen = frozenOf(4, 4);

  it("gives an ended cycle's frozen counts, when it was frozen", () => {
    const row = rowOf(cycleOf("S1", "completed", "2026-10-26", "2026-11-01"), live, frozen);
    expect(cycleCounts(row.status, row, row.progress_snapshot)).toBe(frozen);
  });

  it("gives an ended cycle's live counts until it is frozen", () => {
    const row = rowOf(cycleOf("S1", "completed", "2026-10-26", "2026-11-01"), live);
    expect(cycleCounts(row.status, row, row.progress_snapshot)).toBe(row);
  });

  it.each<CycleStatus>(["current", "upcoming", "draft"])(
    "gives the live counts of a cycle %s, even with counts it was frozen with once (its days changed since)",
    (status) => {
      const row = rowOf(cycleOf("S1", status, status === "draft" ? null : "2026-11-09"), live, frozen);
      expect(cycleCounts(row.status, row, row.progress_snapshot)).toBe(row);
    }
  );

  it("reads a cycle's progress the same way, its distributions and burndown with its counts", () => {
    const now: Progress = { ...live, assignees: [], labels: [], burndown: null };
    const then: FrozenProgress = { ...now, ...countsOf(4, 4), frozen_at: "2026-11-02T00:00:00Z" };
    const ended: CycleProgress = { status: "completed", live: now, progress_snapshot: then };
    const shown: Progress = cycleCounts(ended.status, ended.live, ended.progress_snapshot);
    expect(shown).toBe(then);
    const reopened: CycleProgress = { ...ended, status: "current" };
    expect(cycleCounts(reopened.status, reopened.live, reopened.progress_snapshot)).toBe(now);
  });
});

describe("progressPercent", () => {
  it("is the completed work items of those not cancelled, rounded", () => {
    // 3 of the 8 not cancelled are completed: 37.5 %.
    expect(progressPercent(countsOf(10, 3, 2))).toBe(38);
    expect(progressPercent(countsOf(3, 1))).toBe(33);
    expect(progressPercent(countsOf(3, 2))).toBe(67);
    expect(progressPercent(countsOf(4, 3, 1))).toBe(100);
  });

  it("is 0 without work items, or with every one cancelled", () => {
    expect(progressPercent(countsOf(0, 0))).toBe(0);
    expect(progressPercent(countsOf(3, 0, 3))).toBe(0);
  });
});
