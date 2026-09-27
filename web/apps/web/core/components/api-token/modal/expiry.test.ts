/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, describe, expect, it, vi } from "vitest";
import { expiryDate } from "./expiry";

// When a new token expires (M2 design 7.7). The dates are the local time's, as the page's, so the test holds in
// any time zone, across a change of summer time too.

/** 27 October 2026, 10:20:30, local time: a month of 31 days, then one of 30. */
const now = new Date(2026, 9, 27, 10, 20, 30);

describe("expiryDate", () => {
  it("adds the period chosen to now, by the calendar", () => {
    expect(expiryDate("1_week", now, null)).toEqual(new Date(2026, 10, 3, 10, 20, 30));
    expect(expiryDate("1_month", now, null)).toEqual(new Date(2026, 10, 27, 10, 20, 30));
    expect(expiryDate("3_months", now, null)).toEqual(new Date(2027, 0, 27, 10, 20, 30));
    expect(expiryDate("1_year", now, null)).toEqual(new Date(2027, 9, 27, 10, 20, 30));
  });

  it("ends a token on the day picked at now's time of day, and not before a day is picked", () => {
    expect(expiryDate("custom", now, new Date(2026, 10, 3))).toEqual(new Date(2026, 10, 3, 10, 20, 30));
    expect(expiryDate("custom", now, null)).toBeUndefined();
  });
});

// On the day the clocks change, a day has 25 or 23 hours: the time of day is not the hours since midnight. The
// zone is Berlin's here, whatever the machine's (Node takes a new TZ at once); the dates are built after it is set.
describe("expiryDate on the days summer time ends and starts", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("ends a token on the day picked at now's time of day, and a period keeps it too", () => {
    vi.stubEnv("TZ", "Europe/Berlin");
    // the zone is in force: 25 October 2026 starts in summer time and ends in winter time
    expect(new Date(2026, 9, 25).getTimezoneOffset()).toBe(-120);
    expect(new Date(2026, 9, 25, 12).getTimezoneOffset()).toBe(-60);

    const before = new Date(2026, 9, 24, 10, 20, 30);
    expect(expiryDate("custom", before, new Date(2026, 9, 25))).toEqual(new Date(2026, 9, 25, 10, 20, 30));
    expect(expiryDate("custom", before, new Date(2027, 2, 28))).toEqual(new Date(2027, 2, 28, 10, 20, 30));
    expect(expiryDate("1_week", before, null)).toEqual(new Date(2026, 9, 31, 10, 20, 30));
  });
});
