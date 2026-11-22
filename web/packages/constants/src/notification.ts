/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { NotificationTab } from "@nerve/api-client";

/**
 * The tabs of the notifications page, in their order, each with its label's key (M7 P4 spec 2.6): all, every
 * notification but the mentions; mentions, the mentions alone. Each tab's unread count is nerve's of the same name
 * (NotificationUnreadCount).
 */
export const NOTIFICATION_TABS: readonly { value: NotificationTab; i18n_label: string }[] = [
  { value: "all", i18n_label: "notification.tabs.all" },
  { value: "mentions", i18n_label: "notification.tabs.mentions" },
];
