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

/**
 * The fields a project's intake list is ordered by, in the order menu's order (M7 P5a spec 2.2 C4, 2.7): when a work
 * item was submitted, when it last changed, its number; each with its label's key, whole (tools/i18n-keys.mjs reads
 * it). With a direction, the lowest first or, behind a "-", the highest first, each is one of nerve's six orders.
 */
export const INTAKE_ORDER_FIELDS: readonly {
  value: "created_at" | "updated_at" | "sequence_id";
  i18n_label: string;
}[] = [
  { value: "created_at", i18n_label: "inbox_issue.order_by.created_at" },
  { value: "updated_at", i18n_label: "inbox_issue.order_by.updated_at" },
  { value: "sequence_id", i18n_label: "inbox_issue.order_by.id" },
];

/**
 * The days a project's intake list is filtered by in a press, by when its work items were submitted or last changed
 * (M7 P5a spec 2.2 C3, R10): today, yesterday, the last 7 or 30 days, each with its label's key, whole. The page turns
 * each into days of the caller's calendar as it is chosen.
 */
export const INTAKE_DATE_PRESETS: readonly {
  value: "today" | "yesterday" | "last_7_days" | "last_30_days";
  i18n_label: string;
}[] = [
  { value: "today", i18n_label: "inbox_issue.filters.presets.today" },
  { value: "yesterday", i18n_label: "inbox_issue.filters.presets.yesterday" },
  { value: "last_7_days", i18n_label: "inbox_issue.filters.presets.last_7_days" },
  { value: "last_30_days", i18n_label: "inbox_issue.filters.presets.last_30_days" },
];
