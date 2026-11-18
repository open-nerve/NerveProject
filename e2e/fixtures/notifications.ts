import type { components, operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

// The notifications' calls of the stories (M7 design 2, 5; P1c spec 2.2): each the caller's own, of a workspace by its
// slug, or one of his by its id.

export type Notification = components["schemas"]["Notification"];
type NotificationPage = components["schemas"]["NotificationPage"];
type NotificationQuery = NonNullable<operations["listNotifications"]["parameters"]["query"]>;
type NotificationUpdate = components["schemas"]["NotificationUpdate"];
type NotificationMarkAllRead = components["schemas"]["NotificationMarkAllRead"];

/**
 * The answer of GET /api/v0/workspaces/{slug}/notifications by the caller of token: its status, the page, or the
 * problem's code and fields.
 */
export async function notificationsListed(api: Api, token: string, slug: string, query: NotificationQuery = {}) {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/notifications", {
    params: { path: { slug }, query },
    headers: bearer(token),
  });
  return data ? { status: response.status, page: data } : answer(response, error);
}

/** The page of the caller of token's notifications in the workspace of slug that query asks for. */
export async function listNotifications(
  api: Api,
  token: string,
  slug: string,
  query: NotificationQuery = {}
): Promise<NotificationPage> {
  const listed = await notificationsListed(api, token, slug, query);
  expect(listed.status, `the notifications in ${slug} ${JSON.stringify(query)}: ${JSON.stringify(listed)}`).toBe(200);
  if (!("page" in listed)) {
    throw new Error(`listNotifications in ${slug} answered 200 without a page`);
  }
  return listed.page;
}

/**
 * Waits, a bounded while, for the caller of token to be listed at least count notifications in the workspace of slug
 * as query selects them, and answers them: nerve sends a write's notifications from its notification.fanout job, all
 * of one write's in one insert, after the write has answered (M7 P1b spec 2.9). The story then says which they are.
 */
export async function notificationsSent(
  api: Api,
  token: string,
  slug: string,
  count: number,
  query: NotificationQuery = {}
): Promise<Notification[]> {
  let sent: Notification[] = [];
  await expect
    .poll(
      async () => {
        sent = (await listNotifications(api, token, slug, query)).data;
        return sent.length;
      },
      {
        timeout: 10_000,
        intervals: [20, 50, 100],
        message: `${count} notifications in ${slug} ${JSON.stringify(query)}`,
      }
    )
    .toBeGreaterThanOrEqual(count);
  return sent;
}

/** The caller of token's unread notifications in the workspace of slug, as GET …/notifications/unread-count counts them. */
export async function unreadNotifications(api: Api, token: string, slug: string) {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/notifications/unread-count", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `the unread notifications in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`getNotificationUnreadCount in ${slug} answered 200 without the counts`);
  }
  return data;
}

/**
 * The answer of PATCH /api/v0/notifications/{notification_id} by the caller of token: its status, the notification,
 * or the problem's code and fields.
 */
export async function notificationUpdated(api: Api, token: string, notificationId: string, body: NotificationUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/notifications/{notification_id}", {
    params: { path: { notification_id: notificationId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, notification: data } : answer(response, error);
}

/**
 * The answer of POST /api/v0/workspaces/{slug}/notifications/mark-all-read by the caller of token: its status and how
 * many it marked, or the problem's code and fields.
 */
export async function allNotificationsMarked(api: Api, token: string, slug: string, body: NotificationMarkAllRead) {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/notifications/mark-all-read", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, marked: data.marked } : answer(response, error);
}
