/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The notifications of the tests of the notification store and its readers (P4 spec 2.3), against a fake nerve: the
// caller's notifications and unread counts as nerve gives them, the requests of the notifications, a tab whose caller
// nerve listed acme and beta, and the fetches that load a workspace's list or count into the store.

import type { Notification, NotificationPage, NotificationUnreadCount, Workspace } from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { fakeRoot, withoutImageObservation } from "@/store/fake-root";
import type { NotificationFilters } from "@/store/notification/filters";
import type { INotificationStore } from "@/store/notification/notification.store";
import { NotificationStore } from "@/store/notification/notification.store";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

export const acme = workspaceOf("acme");
export const beta = workspaceOf("beta");

/**
 * One of the caller's notifications in acme as nerve lists it, its id named by name: unread, neither archived nor
 * snoozed, sent to him as WEB-1's subscriber when Ann renamed it, unless fields say so.
 */
export function notificationOf(name: string, fields: Partial<Notification> = {}): Notification {
  return {
    id: `n-${name}`,
    workspace_id: acme.id,
    project_id: "p-web",
    issue_id: "i-web-1",
    sender: "subscribed",
    triggered_by_id: "u-ann",
    data: {
      issue: {
        id: "i-web-1",
        name: "Sign-in",
        sequence_id: 1,
        identifier: "WEB",
        state_name: "Todo",
        state_group: "unstarted",
      },
      change: {
        field: "name",
        verb: "updated",
        old_value: "Login",
        new_value: "Sign-in",
        old_id: null,
        new_id: null,
        old_issue: null,
        new_issue: null,
        comment_id: null,
        excerpt: null,
      },
    },
    read_at: null,
    archived_at: null,
    snoozed_till: null,
    created_at: "2026-11-01T09:00:00Z",
    ...fields,
  };
}

/** A page of the caller's notifications as nerve lists it: the last one unless a cursor follows it. */
export const pageOf = (data: Notification[], nextCursor: string | null = null): NotificationPage => ({
  data,
  next_cursor: nextCursor,
});

/** The caller's unread notifications in a workspace as nerve counts them: all is every one but the mentions. */
export const countOf = (all: number, mentions: number): NotificationUnreadCount => ({ all, mentions });

/** The requests of the notifications: a workspace's list and unread counts, by its slug. */
export const notificationRequests = {
  list: (workspace: Pick<Workspace, "slug">) => `/api/v0/workspaces/${workspace.slug}/notifications`,
  unreadCount: (workspace: Pick<Workspace, "slug">) =>
    `/api/v0/workspaces/${workspace.slug}/notifications/unread-count`,
};

/**
 * A notification store on the stores of a tab at acme's address, whose caller's acme and beta nerve listed; the
 * requests so far are forgotten. The router store is the tab's, for a test whose route names another workspace.
 */
export async function notificationTab() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(
    fakeRoot({ observeProfileRead: withoutImageObservation, router }),
    api,
    () => () => true
  );
  router.setQuery({ workspaceSlug: acme.slug });
  await loadWorkspaces(nerve, workspaceRoot, [acme, beta]);
  nerve.calls.length = 0;
  const store = new NotificationStore(fakeRoot({ workspaceRoot }), api);
  return { nerve, api, router, workspaceRoot, store };
}

/** The store fetches the workspace's unread counts, and nerve gives count. */
export function loadUnreadCount(
  nerve: FakeNerve,
  store: INotificationStore,
  workspace: Pick<Workspace, "id" | "slug">,
  count: NotificationUnreadCount
) {
  const path = notificationRequests.unreadCount(workspace);
  return answered(nerve, () => store.fetchUnreadCount(workspace), ["GET", path], count, "the count");
}

/** The store fetches the first page of the workspace's notifications the filters select, and nerve gives page. */
export function loadList(
  nerve: FakeNerve,
  store: INotificationStore,
  workspace: Pick<Workspace, "id" | "slug">,
  filters: NotificationFilters,
  page: NotificationPage
) {
  const path = notificationRequests.list(workspace);
  return answered(nerve, () => store.fetchList(workspace, filters), ["GET", path], page, "the list");
}
