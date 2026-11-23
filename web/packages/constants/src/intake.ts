/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IntakeStatus } from "@nerve/api-client";
import type { TInboxIssueStatus } from "@nerve/types";
import { EInboxIssueStatus } from "@nerve/types";

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

export const INBOX_STATUS: {
  key: string;
  status: TInboxIssueStatus;
  i18n_title: string;
  i18n_description: () => string;
}[] = [
  {
    key: "pending",
    i18n_title: "inbox_issue.status.pending.title",
    status: EInboxIssueStatus.PENDING,
    i18n_description: () => `inbox_issue.status.pending.description`,
  },
  {
    key: "declined",
    i18n_title: "inbox_issue.status.declined.title",
    status: EInboxIssueStatus.DECLINED,
    i18n_description: () => `inbox_issue.status.declined.description`,
  },
  {
    key: "snoozed",
    i18n_title: "inbox_issue.status.snoozed.title",
    status: EInboxIssueStatus.SNOOZED,
    i18n_description: () => `inbox_issue.status.snoozed.description`,
  },
  {
    key: "accepted",
    i18n_title: "inbox_issue.status.accepted.title",
    status: EInboxIssueStatus.ACCEPTED,
    i18n_description: () => `inbox_issue.status.accepted.description`,
  },
  {
    key: "duplicate",
    i18n_title: "inbox_issue.status.duplicate.title",
    status: EInboxIssueStatus.DUPLICATE,
    i18n_description: () => `inbox_issue.status.duplicate.description`,
  },
];

export const INBOX_ISSUE_ORDER_BY_OPTIONS = [
  {
    key: "issue__created_at",
    i18n_label: "inbox_issue.order_by.created_at",
  },
  {
    key: "issue__updated_at",
    i18n_label: "inbox_issue.order_by.updated_at",
  },
  {
    key: "issue__sequence_id",
    i18n_label: "inbox_issue.order_by.id",
  },
];

export const INBOX_ISSUE_SORT_BY_OPTIONS = [
  {
    key: "asc",
    i18n_label: "common.sort.asc",
  },
  {
    key: "desc",
    i18n_label: "common.sort.desc",
  },
];

export enum EPastDurationFilters {
  TODAY = "today",
  YESTERDAY = "yesterday",
  LAST_7_DAYS = "last_7_days",
  LAST_30_DAYS = "last_30_days",
}

export const PAST_DURATION_FILTER_OPTIONS: {
  name: string;
  value: string;
}[] = [
  {
    name: "Today",
    value: EPastDurationFilters.TODAY,
  },
  {
    name: "Yesterday",
    value: EPastDurationFilters.YESTERDAY,
  },
  {
    name: "Last 7 days",
    value: EPastDurationFilters.LAST_7_DAYS,
  },
  {
    name: "Last 30 days",
    value: EPastDurationFilters.LAST_30_DAYS,
  },
];
