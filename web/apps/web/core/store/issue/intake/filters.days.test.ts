/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, describe, expect, it, vi } from "vitest";
import { open } from "./fake-intake";
import type { IntakeFilters } from "./filters";
import {
  customRange,
  customized,
  defaultFilters,
  listQuery,
  presetOf,
  presetRange,
  tabStatuses,
  toggledStatuses,
} from "./filters";

// What the intake page's filters menu and its applied filters change (P5a spec 2.2, R9, R10; the contract's C1, C3):
// the statuses a tab lists, one at least, the tab's own alone; the days a preset names, days of the caller's calendar
// as nerve takes days (a day, a..b), at the moment it is chosen, wherever the caller is; the days the custom dates'
// dialog chose (after a day, before a day, between two) as nerve's a.., ..b and a..b; and whether the filters list
// other work items than the tab's first ones, the order aside (an order lists the same).

afterEach(() => vi.unstubAllEnvs());

describe("tabStatuses", () => {
  it("lists the open tab's pending and snoozed, the closed tab's three final statuses (C1)", () => {
    expect(tabStatuses("open")).toEqual(["pending", "snoozed"]);
    expect(tabStatuses("closed")).toEqual(["accepted", "declined", "duplicate"]);
  });
});

describe("toggledStatuses", () => {
  it("turns a status on after those held, or off if held, the others kept", () => {
    expect(toggledStatuses(["pending"], "snoozed")).toEqual(["pending", "snoozed"]);
    expect(toggledStatuses(["pending", "snoozed"], "pending")).toEqual(["snoozed"]);
    expect(toggledStatuses(["accepted", "declined", "duplicate"], "declined")).toEqual(["accepted", "duplicate"]);
  });

  it("keeps the last status on: none, so that the list names a status still (R9; C1: none named, every status)", () => {
    expect(toggledStatuses(["pending"], "pending")).toBeUndefined();
    expect(toggledStatuses(["duplicate"], "duplicate")).toBeUndefined();
  });
});

describe("customized", () => {
  it("is false for either tab's first filters, and for an order of the user's own: an order lists the same", () => {
    expect([customized(open), customized(defaultFilters("closed"))]).toEqual([false, false]);
    expect(customized({ ...open, order: "sequence_id" })).toBe(false);
    expect(customized({ ...open, created: {}, updated: {} })).toBe(false);
  });

  it("is true for any filter of the user's own: a status more or less, a value named, a range of days", () => {
    const cases: Partial<IntakeFilters>[] = [
      { statuses: ["pending", "snoozed"] },
      { statuses: ["snoozed"] },
      { priorities: ["urgent"] },
      { assignees: ["u-ann"] },
      { creators: ["u-bob"] },
      { labels: ["l-bug"] },
      { created: { from: "2026-11-01" } },
      { updated: { to: "2026-11-01" } },
    ];
    expect(cases.map((fields) => customized({ ...open, ...fields }))).toEqual(cases.map(() => true));
  });
});

describe("presetRange", () => {
  // half an hour before the caller's midnight west of UTC, whose UTC day is already the next; half an hour after
  // seven in the morning east of it, whose UTC day is still the one before
  it.each([
    {
      zone: "America/Los_Angeles",
      now: "2026-11-24T07:30:00Z",
      days: {
        today: { from: "2026-11-23", to: "2026-11-23" },
        yesterday: { from: "2026-11-22", to: "2026-11-22" },
        last_7_days: { from: "2026-11-16", to: "2026-11-23" },
        // across the end of daylight saving time (1 November)
        last_30_days: { from: "2026-10-24", to: "2026-11-23" },
      },
    },
    {
      zone: "Asia/Shanghai",
      now: "2026-11-23T23:30:00Z",
      days: {
        today: { from: "2026-11-24", to: "2026-11-24" },
        yesterday: { from: "2026-11-23", to: "2026-11-23" },
        last_7_days: { from: "2026-11-17", to: "2026-11-24" },
        last_30_days: { from: "2026-10-25", to: "2026-11-24" },
      },
    },
  ])("names days of the caller's calendar at now, as nerve takes days ($zone, R10)", ({ zone, now, days }) => {
    vi.stubEnv("TZ", zone);
    const at = new Date(now);
    const presets = ["today", "yesterday", "last_7_days", "last_30_days"] as const;
    expect(Object.fromEntries(presets.map((preset) => [preset, presetRange(preset, at)]))).toEqual(days);
    // a day alone for today and yesterday, a..b for the others (C3)
    expect(presets.map((preset) => listQuery({ ...open, created: presetRange(preset, at) }).created_at)).toEqual([
      days.today.from,
      days.yesterday.from,
      `${days.last_7_days.from}..${days.last_7_days.to}`,
      `${days.last_30_days.from}..${days.last_30_days.to}`,
    ]);
  });
});

describe("presetOf", () => {
  it("names the preset whose days the range is at now, as now is: none for other days, nor for no range", () => {
    vi.stubEnv("TZ", "America/Los_Angeles");
    const now = new Date("2026-11-23T20:00:00Z");
    const tomorrow = new Date("2026-11-24T20:00:00Z");
    expect(presetOf(presetRange("last_7_days", now), now)).toBe("last_7_days");
    expect(presetOf(presetRange("yesterday", now), now)).toBe("yesterday");
    // a day on, today's day is yesterday, and the last 7 days are no preset's
    expect(presetOf(presetRange("today", now), tomorrow)).toBe("yesterday");
    expect(presetOf(presetRange("last_7_days", now), tomorrow)).toBeUndefined();
    expect(presetOf({ from: "2026-11-01" }, now)).toBeUndefined();
    expect(presetOf(undefined, now)).toBeUndefined();
  });
});

describe("customRange", () => {
  it("takes the dialog's after, before and both as nerve's a.., ..b and a..b (C3): each day in, one range", () => {
    const sent = (values: string[]) => listQuery({ ...open, updated: customRange(values) }).updated_at;
    expect(customRange(["2026-11-01;after"])).toEqual({ from: "2026-11-01" });
    expect(customRange(["2026-11-20;before"])).toEqual({ to: "2026-11-20" });
    expect(customRange(["2026-11-01;after", "2026-11-20;before"])).toEqual({ from: "2026-11-01", to: "2026-11-20" });
    expect([
      sent(["2026-11-01;after"]),
      sent(["2026-11-20;before"]),
      sent(["2026-11-01;after", "2026-11-20;before"]),
      // a range of one day is that day
      sent(["2026-11-05;after", "2026-11-05;before"]),
    ]).toEqual(["2026-11-01..", "..2026-11-20", "2026-11-01..2026-11-20", "2026-11-05"]);
  });
});
