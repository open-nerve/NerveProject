/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { IntakeStatus, IssueIntake } from "@nerve/api-client";
import { detailOf, intakeIssueOf } from "./fake-intake";
import { effectiveStatus, rowOf } from "./status";

// The status of a work item of an intake and the row the intake list shows of it (P5a spec 2.2; the contract's trap
// C7): nerve answers, lists and counts each status at the moment of the request, a snooze whose time has come as
// pending (its snoozed_till not after the moment); a list's row is the work item without its HTML and the caller's
// subscription, and a work item never submitted has none.

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
