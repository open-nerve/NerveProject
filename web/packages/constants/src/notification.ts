/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { NotificationTab, NotificationType } from "@nerve/api-client";

/**
 * The tabs of the notifications page, in their order, each with its label's key (M7 P4 spec 2.6): all, every
 * notification but the mentions; mentions, the mentions alone. Each tab's unread count is nerve's of the same name
 * (NotificationUnreadCount).
 */
export const NOTIFICATION_TABS: readonly { value: NotificationTab; i18n_label: string }[] = [
  { value: "all", i18n_label: "notification.tabs.all" },
  { value: "mentions", i18n_label: "notification.tabs.mentions" },
];

/**
 * The senders the notifications page's all tab filters by, in the order nerve is asked for them (M7 P4 spec 2.2,
 * 2.6), each with its label's key: why nerve notified the caller, each NotificationType once.
 */
export const NOTIFICATION_TYPE_OPTIONS: readonly { value: NotificationType; i18n_label: string }[] = [
  { value: "assigned", i18n_label: "notification.filter.assigned" },
  { value: "created", i18n_label: "notification.filter.created" },
  { value: "subscribed", i18n_label: "notification.filter.subscribed" },
];
