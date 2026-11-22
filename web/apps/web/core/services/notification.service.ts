/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, NotificationUnreadCount } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * The caller's notifications (M7 design 3.10, P4 spec 2.1): each operation of nerve's, its answer as nerve gives it.
 */
export class NotificationService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

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
