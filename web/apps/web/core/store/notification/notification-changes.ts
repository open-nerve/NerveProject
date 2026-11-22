/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
// nerve imports
import type {
  Notification,
  NotificationPage,
  NotificationsMarked,
  NotificationUnreadCount,
  NotificationUpdate,
  Workspace,
} from "@nerve/api-client";
// lib
import { problemCode } from "@/lib/api-error";
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change, ReconciledByKey } from "@/lib/reconciled";
import { dropped, replaced } from "@/lib/reconciled";
// services
import type { NotificationService } from "@/services/notification.service";
// local
import type { NotificationFilters } from "./filters";
import { markAllBody, matches } from "./filters";

/** A workspace as the store names it: by its slug, its values kept under its id. */
export type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface INotificationChanges {
  // changes
  update: (workspace: WorkspaceRef, notificationId: string, patch: NotificationUpdate) => Promise<Notification>;
  markAllRead: (workspace: WorkspaceRef, filters: NotificationFilters) => Promise<NotificationsMarked>;
}

/** A list the store fetched: whose workspace it is, and the filters its key names (filtersOf). */
export type FetchedList = { workspaceId: string; filters: NotificationFilters };

/** What the changes write nerve's answers into and fetch again: the notification store's (NotificationStore). */
type HeldNotifications = {
  /** Each workspace's lists, by their keys. */
  readonly lists: ReconciledByKey<NotificationPage>;
  /** The lists the store fetched, by their keys. */
  readonly fetched: ReadonlyMap<string, FetchedList>;
  fetchUnreadCount: (workspace: WorkspaceRef) => Promise<NotificationUnreadCount | undefined>;
};

/**
 * The page with the notification as nerve answered it, at now: in its place where the page's filters still list it
 * (its read state never moves it, R8), out of the page where they no longer do; never into a page that newly would
 * list it, which its next fetch brings.
 */
const written =
  (notification: Notification, filters: NotificationFilters, now: Date): Change<NotificationPage> =>
  (page) => ({
    ...page,
    data: matches(notification, filters, now)
      ? replaced(notification)(page.data)
      : dropped<Notification>(notification.id)(page.data),
  });

/** The page without the notification id names. */
const without =
  (notificationId: string): Change<NotificationPage> =>
  (page) => ({ ...page, data: dropped<Notification>(notificationId)(page.data) });

/**
 * The changes of the caller's notifications (P4 spec 2.3, R6, R8, R9), one at a time (v0 design 7.7). Once nerve
 * confirms a change, its answer is written into every list the store holds of the workspace (written); a notification
 * nerve no longer gives him (notification.not_found) leaves each of them. Then the workspace's unread count is fetched
 * again: only nerve counts it. Mark all read writes nothing here: the count is fetched again, whether nerve marked or
 * refused (it marks in batches, and those before a failed one stay marked), and the page reads the list's first page
 * again through the list's own read. The fetches again do not hold the queue; one that fails leaves what the store
 * had. A refusal fails the change, the values as they were but for notification.not_found, and goes to the caller.
 */
export class NotificationChanges implements INotificationChanges {
  /** The changes of the caller's notifications, sent one at a time. */
  private readonly queue = oneAtATime();

  constructor(
    private readonly held: HeldNotifications,
    private readonly service: NotificationService
  ) {}

  /**
   * @description changes the notification's states patch names (read, archived, snoozed_till: null ends its snooze);
   * the workspace's lists then show nerve's answer, and its count is fetched again. Fails when nerve refuses: for
   * notification.not_found the notification leaves every list of the workspace and the count is fetched again, for any
   * other refusal nothing changes.
   */
  update = (workspace: WorkspaceRef, notificationId: string, patch: NotificationUpdate): Promise<Notification> =>
    this.queue(async () => {
      const notification = await this.service.update(notificationId, patch).catch((error: unknown) => {
        if (problemCode(error) === "notification.not_found") {
          this.confirm(workspace, () => without(notificationId));
          this.countAgain(workspace);
        }
        throw error;
      });
      const now = new Date();
      this.confirm(workspace, (filters) => written(notification, filters, now));
      this.countAgain(workspace);
      return notification;
    });

  /**
   * @description marks read the workspace's unread notifications the filters select (markAllBody, in its turn); gives
   * how many nerve marked. No row is marked here, nor any count zeroed: the count is fetched again, whether nerve
   * marked them or refused, and a refusal then fails it. The filters' first page is the page's to read again, either
   * way, through the list's own read (the notifications page's head), so that the list it shows has it.
   */
  markAllRead = (workspace: WorkspaceRef, filters: NotificationFilters): Promise<NotificationsMarked> =>
    this.queue(async () => {
      try {
        return await this.service.markAllRead(workspace.slug, markAllBody(filters));
      } finally {
        this.countAgain(workspace);
      }
    });

  /** Confirms to every list the store holds of the workspace the change made of its filters. */
  private confirm(workspace: WorkspaceRef, change: (filters: NotificationFilters) => Change<NotificationPage>) {
    runInAction(() => {
      for (const [key, list] of this.held.fetched)
        if (list.workspaceId === workspace.id) this.held.lists.confirm(key, change(list.filters));
    });
  }

  /** Fetches the workspace's unread count again, not holding the queue; a failure leaves the count as it was. */
  private countAgain(workspace: WorkspaceRef) {
    void Promise.allSettled([this.held.fetchUnreadCount(workspace)]);
  }
}
