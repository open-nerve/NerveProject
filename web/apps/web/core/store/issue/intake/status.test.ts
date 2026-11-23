/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeStatus, IssueIntake } from "@nerve/api-client";
import { detailOf, intakeIssueOf } from "./fake-intake";
import { effectiveStatus, inIntake, rowOf, snoozeDay, snoozeInstant } from "./status";

// The status of a work item of an intake and the row the intake list shows of it (P5a spec 2.2; the contract's trap
// C7): nerve answers, lists and counts each status at the moment of the request, a snooze whose time has come as
// pending (its snoozed_till not after the moment); a list's row is the work item without its HTML and the caller's
// subscription, and a work item never submitted has none. A work item is in its intake while its detail's intake row is
// not accepted; one held as a list's row alone is not (lists list none in an intake). A snooze is till a day of the
// caller's calendar, tomorrow at the earliest, from its first moment in the browser's zone (R13).

const NOW = new Date("2026-11-10T09:00:00Z");
const intake = (status: IntakeStatus, snoozedTill: string | null = null): IssueIntake => ({
  status,
  snoozed_till: snoozedTill,
  duplicate_to_id: null,
});

describe("effectiveStatus", () => {
  it("is pending for a snooze whose time has come, snoozed for one whose time has not (C7)", () => {
    expect(effectiveStatus(intake("snoozed", "2026-11-10T08:59:59.999Z"), NOW)).toBe("pending");
    expect(effectiveStatus(intake("snoozed", "2026-11-09T09:00:00Z"), NOW)).toBe("pending");
    expect(effectiveStatus(intake("snoozed", "2026-11-10T09:00:00.001Z"), NOW)).toBe("snoozed");
    expect(effectiveStatus(intake("snoozed", "2026-11-11T00:00:00Z"), NOW)).toBe("snoozed");
  });

  it("is pending at the very moment the snooze ends: nerve's snoozed_till <= now", () => {
    expect(effectiveStatus(intake("snoozed", "2026-11-10T09:00:00Z"), NOW)).toBe("pending");
    // the same moment in another offset is the same moment
    expect(effectiveStatus(intake("snoozed", "2026-11-10T17:00:00+08:00"), NOW)).toBe("pending");
  });

  it.each<IntakeStatus>(["pending", "accepted", "declined", "duplicate"])(
    "is %s for itself, whatever the moment",
    (status) => {
      expect(effectiveStatus(intake(status), NOW)).toBe(status);
      expect(effectiveStatus(intake(status), new Date("2030-01-01T00:00:00Z"))).toBe(status);
    }
  );
});

describe("rowOf", () => {
  it("is the work item as the intake list shows it: its row's keys alone, no HTML, no subscription", () => {
    const snoozed = intake("snoozed", "2026-11-11T00:00:00Z");
    const detail = detailOf("one", { name: "Sign-in fails", is_subscribed: true, intake: snoozed, label_ids: ["l-1"] });
    expect(rowOf(detail)).toStrictEqual(
      intakeIssueOf("one", { name: "Sign-in fails", intake: snoozed, label_ids: ["l-1"] })
    );
    expect(Object.keys(rowOf(detail) ?? {})).not.toContain("description_html");
  });

  it("is a row for a work item out of the intake by a triage too: the closed tab lists it", () => {
    const accepted = intake("accepted");
    expect(rowOf(detailOf("two", { intake: accepted, state_id: "s-todo" }))).toStrictEqual(
      intakeIssueOf("two", { intake: accepted, state_id: "s-todo" })
    );
  });

  it("is none for a work item never submitted to its project's intake (intake null): no intake lists it", () => {
    expect(rowOf(detailOf("three", { intake: null }))).toBeUndefined();
  });
});

describe("inIntake", () => {
  it.each<IntakeStatus>(["pending", "snoozed", "declined", "duplicate"])(
    "is true for a detail whose intake row is %s: its changes go to its intake",
    (status) => {
      expect(inIntake(detailOf("one", { intake: intake(status) }))).toBe(true);
    }
  );

  it("is false for an accepted detail, one never submitted (intake null), and a work item not held as a detail", () => {
    expect(inIntake(detailOf("one", { intake: intake("accepted") }))).toBe(false);
    expect(inIntake(detailOf("one", { intake: null }))).toBe(false);
    expect(inIntake(undefined)).toBe(false);
  });
});

/** A moment as the browser's zone reads it: its year, month (from 1), day and hour. */
const day = (date: Date) => [date.getFullYear(), date.getMonth() + 1, date.getDate(), date.getHours()];

describe("snoozeDay", () => {
  // 25 November 2026, 23:30 in the browser's zone: a minute before midnight is today still
  const now = new Date(2026, 10, 25, 23, 30);

  it("is tomorrow, at its start, for a snooze not held: the earliest day and the first (R13)", () => {
    expect(day(snoozeDay(now))).toEqual([2026, 11, 26, 0]);
    expect(day(snoozeDay(new Date(2026, 11, 31, 0, 0)))).toEqual([2027, 1, 1, 0]);
  });

  it("is the held snooze's day while it is tomorrow or later, else tomorrow: a day gone by is no choice", () => {
    expect(day(snoozeDay(now, new Date(2026, 10, 30, 17, 0).toISOString()))).toEqual([2026, 11, 30, 0]);
    expect(day(snoozeDay(now, new Date(2026, 10, 26, 0, 0).toISOString()))).toEqual([2026, 11, 26, 0]);
    expect(day(snoozeDay(now, new Date(2026, 10, 25, 23, 59).toISOString()))).toEqual([2026, 11, 26, 0]);
    expect(day(snoozeDay(now, null))).toEqual([2026, 11, 26, 0]);
  });
});

describe("snoozeInstant", () => {
  // the zone of the process, which Date reads (vi.stubEnv); the CI's zone is UTC, where it does not show
  afterEach(() => vi.unstubAllEnvs());

  describe("in America/Los_Angeles", () => {
    beforeEach(() => vi.stubEnv("TZ", "America/Los_Angeles"));

    it("runs west of UTC", () => {
      expect(new Date(2026, 10, 25).getTimezoneOffset()).toBe(480);
    });

    it("is the day's first moment there, as nerve takes a moment: midnight in Los Angeles, not in UTC (R13)", () => {
      expect(snoozeInstant(new Date(2026, 10, 25))).toBe("2026-11-25T08:00:00.000Z");
      // whatever moment of the day the calendar gives
      expect(snoozeInstant(new Date(2026, 10, 25, 13, 45))).toBe("2026-11-25T08:00:00.000Z");
    });
  });

  describe("in America/Santiago, whose clocks skip midnight as daylight time starts (6 September 2026)", () => {
    beforeEach(() => vi.stubEnv("TZ", "America/Santiago"));
    const santiago = new Intl.DateTimeFormat("en-CA", {
      timeZone: "America/Santiago",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
    });

    it("is the day's first moment there that exists: 01:00, the clocks going from 23:59:59 to 01:00", () => {
      const moment = snoozeInstant(new Date(2026, 8, 6, 12, 0));
      expect(moment).toBe("2026-09-06T04:00:00.000Z");
      expect(santiago.format(new Date(moment))).toBe("2026-09-06, 01:00");
      // the day before and the day after start at midnight
      expect(santiago.format(new Date(snoozeInstant(new Date(2026, 8, 5, 12, 0))))).toBe("2026-09-05, 00:00");
      expect(santiago.format(new Date(snoozeInstant(new Date(2026, 8, 7, 12, 0))))).toBe("2026-09-07, 00:00");
    });
  });
});
