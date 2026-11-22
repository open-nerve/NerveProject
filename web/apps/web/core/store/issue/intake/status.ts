/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IntakeIssue, IntakeStatus, IssueDetail, IssueIntake } from "@nerve/api-client";

/**
 * @description the status of the intake row at now, as nerve answers, lists and counts it (C7): a snooze whose time
 * has come, its snoozed_till not after now, is pending; any other status is its own
 */
export const effectiveStatus = (intake: IssueIntake, now: Date): IntakeStatus =>
  intake.status === "snoozed" &&
  intake.snoozed_till !== null &&
  new Date(intake.snoozed_till).getTime() <= now.getTime()
    ? "pending"
    : intake.status;

/**
 * @description the work item nerve answered as the intake list shows it (IntakeIssue): the keys of the row alone, not
 * its HTML or the caller's subscription, which a list's row does not carry; none for a work item never submitted to
 * its project's intake, or whose intake row is gone (intake null), which no intake list lists
 */
export function rowOf(detail: IssueDetail): IntakeIssue | undefined {
  const { description_html, is_subscribed, intake, ...row } = detail;
  return intake === null ? undefined : { ...row, intake };
}
