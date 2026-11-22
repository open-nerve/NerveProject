/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { SNOOZE_PRESETS, presetMoment, slotLabel, slotMoment, snoozeSlots } from "./snooze-time";

// The times a notification is snoozed till (P4 spec 2.2, R15): the menu's presets, of 1, 3, 5, 7 and 14 days, each
// from the moment it is chosen, days of 24 hours, never from the day's start nor by the calendar's days; the custom
// dialog's half hours of a day, "HH:MM" of 24 hours, today's those after now alone, each the moment of the day in the
// browser's zone (so that 00:30 is half past midnight and 12:30 half past noon, W1), said in the page's
// language: 12 hours in English, 24 in Chinese.

describe("SNOOZE_PRESETS", () => {
  it("are 1, 3, 5, 7 and 14 days, each by its words", () => {
    expect(SNOOZE_PRESETS).toEqual([
      { days: 1, i18n_label: "notification.snooze.1_day" },
      { days: 3, i18n_label: "notification.snooze.3_days" },
      { days: 5, i18n_label: "notification.snooze.5_days" },
      { days: 7, i18n_label: "notification.snooze.1_week" },
      { days: 14, i18n_label: "notification.snooze.2_weeks" },
    ]);
  });
});

describe("presetMoment", () => {
  const now = new Date("2026-11-20T15:34:56.789Z");

  it("is now and the days, as nerve takes a moment", () => {
    expect(presetMoment(1, now)).toBe("2026-11-21T15:34:56.789Z");
    expect(presetMoment(3, now)).toBe("2026-11-23T15:34:56.789Z");
    expect(presetMoment(14, now)).toBe("2026-12-04T15:34:56.789Z");
  });

  describe("in a zone whose clocks change (America/Los_Angeles)", () => {
    // the zone of the process, which Date reads (vi.stubEnv); the CI's zone is UTC, where it does not show
    beforeAll(() => {
      vi.stubEnv("TZ", "America/Los_Angeles");
    });
    afterAll(() => {
      vi.unstubAllEnvs();
    });

    it("runs west of UTC", () => {
      expect(new Date(2026, 9, 31).getTimezoneOffset()).toBe(420);
    });

    it("adds days of 24 hours, not the calendar's days, across the end of daylight time (1 November 2026)", () => {
      // 31 October, 12:00 in Los Angeles (UTC-7); a calendar day later is 1 November, 12:00 (UTC-8), 25 hours on
      expect(presetMoment(1, new Date("2026-10-31T19:00:00.000Z"))).toBe("2026-11-01T19:00:00.000Z");
    });
  });
});

describe("snoozeSlots", () => {
  const today = new Date(2026, 10, 20);
  const now = new Date(2026, 10, 20, 15, 10);

  it("are the 48 half hours of a day after today, 00:00 to 23:30, and of no day chosen yet", () => {
    const slots = snoozeSlots(new Date(2026, 10, 21), now);
    expect(slots).toHaveLength(48);
    expect(slots.slice(0, 3)).toEqual(["00:00", "00:30", "01:00"]);
    expect(slots.at(-1)).toBe("23:30");
    expect(snoozeSlots(null, now)).toEqual(slots);
  });

  it("are, today, those after now alone, and none past the last", () => {
    expect(snoozeSlots(today, now)).toEqual(snoozeSlots(null, now).slice(31));
    expect(snoozeSlots(today, now)[0]).toBe("15:30");
    expect(snoozeSlots(today, new Date(2026, 10, 20, 15, 30))[0]).toBe("16:00");
    expect(snoozeSlots(today, new Date(2026, 10, 20, 23, 45))).toEqual([]);
  });

  describe("in a zone whose clocks change (America/Los_Angeles, whole-branch m4)", () => {
    beforeAll(() => {
      vi.stubEnv("TZ", "America/Los_Angeles");
    });
    afterAll(() => {
      vi.unstubAllEnvs();
    });

    it("leave out the half hours the start of daylight time skips (14 March 2027): 46, 02:00 and 02:30 none", () => {
      const slots = snoozeSlots(new Date(2027, 2, 14), new Date(2027, 2, 13, 12, 0));
      expect(slots).toHaveLength(46);
      expect(slots.slice(3, 5)).toEqual(["01:30", "03:00"]);
      // each half hour offered is its own moment
      expect(new Set(slots.map((slot) => slotMoment(new Date(2027, 2, 14), slot))).size).toBe(46);
    });

    it("offer the 48 of the day daylight time ends (7 November 2027), 01:00 and 01:30 the first of each", () => {
      const slots = snoozeSlots(new Date(2027, 10, 7), new Date(2027, 10, 6, 12, 0));
      expect(slots).toHaveLength(48);
      expect(slotMoment(new Date(2027, 10, 7), "01:30")).toBe("2027-11-07T08:30:00.000Z");
    });
  });
});

describe("slotMoment", () => {
  it("is the day's half hour in the browser's zone: 00:30 half past midnight, 12:30 half past noon (W1)", () => {
    expect(slotMoment(new Date(2026, 10, 21), "00:30")).toBe(new Date(2026, 10, 21, 0, 30).toISOString());
    expect(slotMoment(new Date(2026, 10, 21), "12:30")).toBe(new Date(2026, 10, 21, 12, 30).toISOString());
  });

  describe("west of UTC (America/Los_Angeles)", () => {
    beforeAll(() => {
      vi.stubEnv("TZ", "America/Los_Angeles");
    });
    afterAll(() => {
      vi.unstubAllEnvs();
    });

    it("is that zone's 00:30 and 12:30, eight hours behind UTC", () => {
      expect(slotMoment(new Date(2026, 10, 21), "00:30")).toBe("2026-11-21T08:30:00.000Z");
      expect(slotMoment(new Date(2026, 10, 21), "12:30")).toBe("2026-11-21T20:30:00.000Z");
    });
  });
});

/** The label, its spaces plain (Intl puts a narrow one before AM and PM). */
const spaced = (label: string) => label.replace(/\s/g, " ");

describe("slotLabel", () => {
  it("says a half hour in 12 hours in English, midnight and noon as such", () => {
    expect(["00:00", "00:30", "12:00", "12:30", "23:30"].map((slot) => spaced(slotLabel(slot, "en")))).toEqual([
      "12:00 AM",
      "12:30 AM",
      "12:00 PM",
      "12:30 PM",
      "11:30 PM",
    ]);
  });

  it("says a half hour in 24 hours in Chinese", () => {
    expect(slotLabel("00:30", "zh-CN")).toMatch(/^0?0:30$/);
    expect(slotLabel("12:30", "zh-CN")).toBe("12:30");
    expect(slotLabel("23:30", "zh-CN")).toBe("23:30");
  });
});
