/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { SNOOZE_PRESETS, presetMoment } from "./snooze-time";

// The times a notification is snoozed till (P4 spec 2.2, R15): the menu's presets, of 1, 3, 5, 7 and 14 days, each
// from the moment it is chosen, days of 24 hours, never from the day's start nor by the calendar's days.

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
