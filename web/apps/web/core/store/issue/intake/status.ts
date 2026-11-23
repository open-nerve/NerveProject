/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IntakeIssue, IntakeStatus, IssueDetail, IssueIntake, Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";

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

/**
 * @description whether the work item is in its project's intake as held (P5a spec 2.2, 2.4): its detail's intake row
 * there and not accepted; not for one held as a list's row alone (undefined here), as lists list none in an intake
 */
export function inIntake(detail: IssueDetail | undefined): boolean {
  const intake = detail?.intake;
  return !!intake && intake.status !== "accepted";
}

/**
 * @description whether the caller may write the work item of a project's intake, its content and its files (P5a spec
 * 2.7, R15, R27; P5b spec 2.2, R9, R25): the project not archived and its intake on; the work item pending or snoozed
 * as stored, as nerve's CheckOpen reads it (a snooze whose time has come is written still; triaged, it is read-only);
 * and the caller with a role in the project (intake.update's three) who is its admin (a workspace admin among its
 * members too, as the permission store gives him) or submitted it, a guest among them. The one rule the intake's
 * detail and a content view of the work item in the intake read
 */
export function intakeEditableBy(
  issue: IssueDetail,
  project: Project | undefined,
  role: EUserPermissions | undefined,
  me: string | undefined
): boolean {
  const status = issue.intake?.status;
  return (
    !!project?.intake_view &&
    !project.archived_at &&
    (status === "pending" || status === "snoozed") &&
    role !== undefined &&
    (role === EUserPermissions.ADMIN || (me !== undefined && issue.created_by_id === me))
  );
}

/**
 * @description the first moment of date's day in the browser's zone: the first that exists, for a day that skips its
 * midnight
 */
const startOf = (date: Date): Date => new Date(date.getFullYear(), date.getMonth(), date.getDate());

/**
 * @description the day the snooze modal starts at (P5a spec 2.8, R13), at its first moment: the held snooze's day while
 * it is tomorrow or later, else tomorrow, the earliest day a snooze is till (nerve refuses a moment not after the
 * request, must_be_future, and any moment of today may be gone by as it is sent); tomorrow alone for a snooze not held
 */
export function snoozeDay(now: Date, held?: string | null): Date {
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
  const day = held ? startOf(new Date(held)) : undefined;
  return day && day.getTime() >= tomorrow.getTime() ? day : tomorrow;
}

/**
 * @description the moment a snooze till day ends, as nerve takes it (R13): the day's first moment in the browser's
 * zone, whatever moment of it the calendar gives; midnight, or, in a zone whose clocks skip it that day, the moment
 * they skip to (the Date of a local time skipped is the moment after the gap, ECMAScript's rule)
 */
export function snoozeInstant(day: Date): string {
  return startOf(day).toISOString();
}
