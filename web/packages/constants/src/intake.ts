/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IntakeStatus } from "@nerve/api-client";

/**
 * The label of each status of a work item in a project's intake, by the status's name as nerve gives it (IntakeStatus;
 * M7 P5a spec 2.2, 2.7): the key of its title, each key whole, so that tools/i18n-keys.mjs reads it. A snooze shows
 * the days it has to go instead (inbox_issue.status.snoozed.description).
 */
export const INTAKE_STATUS_LABELS: Readonly<Record<IntakeStatus, string>> = {
  pending: "inbox_issue.status.pending.title",
  snoozed: "inbox_issue.status.snoozed.title",
  accepted: "inbox_issue.status.accepted.title",
  declined: "inbox_issue.status.declined.title",
  duplicate: "inbox_issue.status.duplicate.title",
};

/**
 * The tabs of a project's intake page, in their order (M7 P5a spec 2.6, 2.7): the work items still to triage, then
 * those triaged, each with its label's key, whole (tools/i18n-keys.mjs reads it). The value is the address's
 * ?currentTab.
 */
export const INTAKE_TABS: readonly { value: "open" | "closed"; i18n_label: string }[] = [
  { value: "open", i18n_label: "inbox_issue.tabs.open" },
  { value: "closed", i18n_label: "inbox_issue.tabs.closed" },
];
