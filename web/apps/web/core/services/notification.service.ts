/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  Notification,
  NotificationMarkAllRead,
  NotificationPage,
  NotificationsMarked,
  NotificationUnreadCount,
  NotificationUpdate,
  operations,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * The query of a page of the caller's notifications, as nerve takes it: each filter a string it parses ("true" or
 * "false", a comma-separated type), the page size a number, the cursor as nerve answered it (filters.ts listQuery).
 */
export type NotificationListQuery = NonNullable<operations["listNotifications"]["parameters"]["query"]>;

/**
 * The caller's notifications (M7 design 3.10, P4 spec 2.1): each operation of nerve's, its answer as nerve gives it.
 */
export class NotificationService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /**
   * A page of the caller's notifications in the workspace slug names, those the query selects, the newest first, with
   * the cursor of the next page (null on the last).
   */
  async list(slug: string, query: NotificationListQuery): Promise<NotificationPage> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/notifications", { params: { path: { slug }, query } }));
  }

  /**
   * Changes the states of the caller's notification body names, each on its own: read, archived, and snoozed_till (a
   * moment after now snoozes it till then, null ends its snooze); the answer is the notification as nerve lists it.
   */
  async update(notificationId: string, body: NotificationUpdate): Promise<Notification> {
    return unwrap(
      await this.api.PATCH("/api/v0/notifications/{notification_id}", {
        params: { path: { notification_id: notificationId } },
        body,
      })
    );
  }

  /**
   * Marks read the caller's unread notifications in the workspace slug names that body's filters select; the answer is
   * how many it marked. nerve marks them in batches: when one fails, those before it stay marked.
   */
  async markAllRead(slug: string, body: NotificationMarkAllRead): Promise<NotificationsMarked> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/notifications/mark-all-read", { params: { path: { slug } }, body })
    );
  }

  /**
   * How many of the caller's notifications in the workspace slug names are unread, not archived and not snoozed now,
   * one count for each tab: all, every one but the mentions, and mentions.
   */
  async unreadCount(slug: string): Promise<NotificationUnreadCount> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/notifications/unread-count", { params: { path: { slug } } })
    );
  }
}
