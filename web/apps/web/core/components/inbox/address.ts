/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IssueIntake } from "@nerve/api-client";
// store
import type { IntakeTab } from "@/store/issue/intake/filters";

/**
 * @description the address of a project's intake page with one of its work items selected (P5b spec 2.4, R14, R25),
 * in the tab that lists the work item's status as stored: the closed tab for one triaged (accepted, declined, a
 * duplicate, as defaultFilters lists them), else the open tab, the page's default (pending, snoozed, a snooze whose
 * time has come too, or no intake row). The one address every opening of a work item of an intake reads
 */
export function intakeAddress(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  intake: IssueIntake | null
): string {
  const tab: IntakeTab = intake && intake.status !== "pending" && intake.status !== "snoozed" ? "closed" : "open";
  return `/${workspaceSlug}/projects/${projectId}/intake?currentTab=${tab}&inboxIssueId=${issueId}`;
}
