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
  Module,
  ModuleListItem,
  ModuleStatus,
  Progress,
} from "@nerve/api-client";
import type { ModuleOrderBy } from "@nerve/constants";
import { MODULE_ORDER_BY_OPTIONS } from "@nerve/constants";
import {
  cycleCounts,
  matchesCycleFilters,
  matchesModuleFilters,
  orderCycleRows,
  orderModuleRows,
  progressPercent,
} from "./planning";

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

// The modules list (P6a spec 2.4, R14): its order, either way, and its filters. A module's progress is
// progressPercent's; its days are days of the project's calendar, compared with the browser's today as days.

/** A module of web, named, with the fields given. */
const moduleOf = (name: string, fields: Partial<Module> = {}): Module => ({
  id: `m-${name}`,
  workspace_id: "w-acme",
  project_id: "p-web",
  name,
  description: "",
  start_date: null,
  target_date: null,
  status: "backlog",
  lead_id: null,
  member_ids: [],
  sort_order: 65535,
  archived_at: null,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
  created_by_id: null,
  updated_by_id: null,
  ...fields,
});
/** A module's row in its list: its fields, and its work items, of which some completed and some cancelled. */
const moduleRowOf = (
  name: string,
  fields: Partial<Module> = {},
  [total, completed, cancelled]: [number, number, number] = [0, 0, 0]
): ModuleListItem => ({ ...moduleOf(name, fields), ...countsOf(total, completed, cancelled) });

describe("orderModuleRows", () => {
  // Four rows as nerve lists them (by sort order), each ordering differently by each option.
  const rows: readonly ModuleListItem[] = [
    moduleRowOf(
      "Pear 10",
      { target_date: "2026-12-01", created_at: "2026-10-02T09:00:00Z", sort_order: 3000 },
      // 2 of the 4 not cancelled: 50 %.
      [10, 2, 6]
    ),
    moduleRowOf(
      "pear 9",
      // Half a second after Apple: nerve writes a moment with the fraction of a second it has, none when it is 0.
      { target_date: null, created_at: "2026-10-01T09:00:00.5Z", sort_order: 1000 },
      [4, 1, 0]
    ),
    moduleRowOf(
      "Apple",
      { target_date: "2026-11-05", created_at: "2026-10-01T09:00:00Z", sort_order: 4000 },
      [10, 7, 0]
    ),
    moduleRowOf("banana", { target_date: "2026-11-20", created_at: "2026-10-03T08:00:00Z", sort_order: 2000 }),
  ];
  // Each option, either way: the manual order has no direction.
  const ORDERS: Record<ModuleOrderBy, string[]> = {
    // Natural: the numbers within the names as numbers, whatever the case.
    name: ["Apple", "banana", "pear 9", "Pear 10"],
    "-name": ["Pear 10", "pear 9", "banana", "Apple"],
    // 0 %, 25 %, 50 %, 70 %: cancelled work items are not counted (the old order, completed and cancelled of all,
    // put Pear 10's 80 % after Apple's 70 %).
    progress: ["banana", "pear 9", "Pear 10", "Apple"],
    "-progress": ["Apple", "Pear 10", "pear 9", "banana"],
    // 0, 4, 10, 10: Pear 10 and Apple have as many and keep nerve's order, either way.
    issues_length: ["banana", "pear 9", "Pear 10", "Apple"],
    "-issues_length": ["Pear 10", "Apple", "pear 9", "banana"],
    // pear 9 has no target day: last, either way.
    target_date: ["Apple", "banana", "Pear 10", "pear 9"],
    "-target_date": ["Pear 10", "banana", "Apple", "pear 9"],
    created_at: ["Apple", "pear 9", "Pear 10", "banana"],
    "-created_at": ["banana", "Pear 10", "pear 9", "Apple"],
    sort_order: ["pear 9", "banana", "Pear 10", "Apple"],
  };
  const options = MODULE_ORDER_BY_OPTIONS.flatMap(({ key }): ModuleOrderBy[] =>
    key === "sort_order" ? [key] : [key, `-${key}`]
  );

  it("has an order for each option of the list's menu, either way, and for none else", () => {
    expect(options.toSorted()).toEqual(Object.keys(ORDERS).toSorted());
  });

  it.each(options)("orders the list by %s", (orderBy) => {
    expect(names(orderModuleRows(rows, orderBy))).toEqual(ORDERS[orderBy]);
  });

  it("keeps the order it was given of rows that order the same, either way", () => {
    const same: readonly ModuleListItem[] = [
      moduleRowOf("beta", { sort_order: 2000 }, [4, 1, 0]),
      moduleRowOf("Alpha", { target_date: "2026-11-05" }, [4, 1, 0]),
      moduleRowOf("alpha", { sort_order: 2000 }, [8, 2, 0]),
      moduleRowOf("gamma", { sort_order: 1000 }, [4, 1, 0]),
    ];
    // Alpha and alpha are one name to the list.
    expect(names(orderModuleRows(same, "name"))).toEqual(["Alpha", "alpha", "beta", "gamma"]);
    expect(names(orderModuleRows(same, "-name"))).toEqual(["gamma", "beta", "Alpha", "alpha"]);
    // 25 % each.
    expect(names(orderModuleRows(same, "progress"))).toEqual(["beta", "Alpha", "alpha", "gamma"]);
    expect(names(orderModuleRows(same, "-progress"))).toEqual(["beta", "Alpha", "alpha", "gamma"]);
    // The modules without a target day after the one with it, as they were given, either way.
    expect(names(orderModuleRows(same, "target_date"))).toEqual(["Alpha", "beta", "alpha", "gamma"]);
    expect(names(orderModuleRows(same, "-target_date"))).toEqual(["Alpha", "beta", "alpha", "gamma"]);
    expect(names(orderModuleRows(same, "sort_order"))).toEqual(["gamma", "beta", "alpha", "Alpha"]);
    // What it was given stays as it was.
    expect(same.map((row) => row.name)).toEqual(["beta", "Alpha", "alpha", "gamma"]);
  });
});

describe("matchesModuleFilters", () => {
  it("filters by nerve's status, any of those chosen; none chosen filters nothing", () => {
    const filters = { status: ["planned", "in-progress"] as ModuleStatus[] };
    expect(matchesModuleFilters(moduleOf("Auth", { status: "planned" }), filters)).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { status: "in-progress" }), filters)).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { status: "paused" }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth", { status: "paused" }), { status: [] })).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { status: "paused" }), {})).toBe(true);
  });

  it("filters by the lead, any of those chosen; a module without a lead has none of them", () => {
    const filters = { lead: ["u-ann", "u-bo"] };
    expect(matchesModuleFilters(moduleOf("Auth", { lead_id: "u-bo" }), filters)).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { lead_id: "u-cy" }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth"), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth"), { lead: [] })).toBe(true);
    // The lead is not a member: a module led by Ann, without members, is not one of Ann's as a member.
    expect(matchesModuleFilters(moduleOf("Auth", { lead_id: "u-ann" }), { members: ["u-ann"] })).toBe(false);
  });

  it("filters by the members, a module with any of those chosen", () => {
    const filters = { members: ["u-ann", "u-bo"] };
    expect(matchesModuleFilters(moduleOf("Auth", { member_ids: ["u-bo", "u-dee"] }), filters)).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { member_ids: ["u-ann"] }), filters)).toBe(true);
    expect(matchesModuleFilters(moduleOf("Auth", { member_ids: ["u-cy", "u-dee"] }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth"), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth", { member_ids: ["u-cy"] }), { members: [] })).toBe(true);
  });

  it("filters by every kind of filter at once", () => {
    const filters = {
      status: ["planned"] as ModuleStatus[],
      lead: ["u-ann"],
      members: ["u-bo"],
      start_date: ["2026-11-01;after"],
    };
    const auth: Partial<Module> = {
      status: "planned",
      lead_id: "u-ann",
      member_ids: ["u-bo"],
      start_date: "2026-11-02",
    };
    expect(matchesModuleFilters(moduleOf("Auth", auth), filters)).toBe(true);
    // Each kind alone leaves it out.
    expect(matchesModuleFilters(moduleOf("Auth", { ...auth, status: "paused" }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth", { ...auth, lead_id: "u-bo" }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth", { ...auth, member_ids: ["u-ann"] }), filters)).toBe(false);
    expect(matchesModuleFilters(moduleOf("Auth", { ...auth, start_date: "2026-10-31" }), filters)).toBe(false);
  });

  // The days, in UTC (the CI's zone) and again in a zone west of UTC, where a "YYYY-MM-DD" read as a time (UTC's
  // midnight) falls on the day before, and today's evening is already tomorrow in UTC: a day read through
  // new Date("YYYY-MM-DD"), or today taken from UTC, is a day off there. The zone is the test's alone (vi.stubEnv).
  describe.each([
    { zone: "UTC", offset: 0, utcToday: "2026-11-11" },
    { zone: "America/Los_Angeles", offset: 480, utcToday: "2026-11-12" },
  ])("by the days a module starts and is due on, in $zone", ({ zone, offset, utcToday }) => {
    beforeAll(() => {
      vi.stubEnv("TZ", zone);
    });
    afterAll(() => {
      vi.unstubAllEnvs();
    });
    // The browser's today: Wednesday 11 November 2026, in the evening.
    beforeEach(() => {
      vi.useFakeTimers({ now: new Date(2026, 10, 11, 21, 30) });
    });
    afterEach(() => {
      vi.useRealTimers();
    });

    it(`runs in ${zone}`, () => {
      expect(new Date(2026, 10, 11).getTimezoneOffset()).toBe(offset);
      expect(new Date().toISOString().slice(0, 10)).toBe(utcToday);
    });

    it.each([
      { option: "1_weeks;after;fromnow", first: "2026-11-18", dayBefore: "2026-11-17" },
      { option: "2_months;after;fromnow", first: "2027-01-10", dayBefore: "2027-01-09" },
    ])("filters $option: a day on $first at the earliest, counting days from today", ({ option, first, dayBefore }) => {
      expect(matchesModuleFilters(moduleOf("Auth", { start_date: first }), { start_date: [option] })).toBe(true);
      expect(matchesModuleFilters(moduleOf("Auth", { start_date: dayBefore }), { start_date: [option] })).toBe(false);
      expect(matchesModuleFilters(moduleOf("Auth", { target_date: first }), { target_date: [option] })).toBe(true);
      expect(matchesModuleFilters(moduleOf("Auth", { target_date: dayBefore }), { target_date: [option] })).toBe(false);
    });

    it("filters by a custom day, the day itself included, each date by its own day", () => {
      const auth = moduleOf("Auth", { start_date: "2026-11-18", target_date: "2026-11-24" });
      expect(matchesModuleFilters(auth, { start_date: ["2026-11-18;after"] })).toBe(true);
      expect(matchesModuleFilters(auth, { start_date: ["2026-11-19;after"] })).toBe(false);
      expect(matchesModuleFilters(auth, { start_date: ["2026-11-18;before"] })).toBe(true);
      expect(matchesModuleFilters(auth, { target_date: ["2026-11-24;before"] })).toBe(true);
      expect(matchesModuleFilters(auth, { target_date: ["2026-11-23;before"] })).toBe(false);
      expect(matchesModuleFilters(auth, { target_date: ["2026-11-24;after"] })).toBe(true);
      // A range of target days, every option chosen: the 20th to the 24th.
      const range = { target_date: ["2026-11-20;after", "2026-11-24;before"] };
      expect(matchesModuleFilters(auth, range)).toBe(true);
      expect(matchesModuleFilters(moduleOf("Auth", { target_date: "2026-11-25" }), range)).toBe(false);
    });

    it("leaves out a module without the day filtered, and keeps it when its days are not filtered", () => {
      const due = moduleOf("Auth", { target_date: "2026-11-24" });
      expect(matchesModuleFilters(due, { start_date: ["2026-11-01;after"] })).toBe(false);
      expect(
        matchesModuleFilters(moduleOf("Auth", { start_date: "2026-11-02" }), { target_date: ["2027-01-01;before"] })
      ).toBe(false);
      expect(matchesModuleFilters(moduleOf("Auth"), { start_date: [], target_date: [] })).toBe(true);
    });
  });
});
