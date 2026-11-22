/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The notifications of the tests of the notification store and its readers (P4 spec 2.3), against a fake nerve: the
// caller's unread counts as nerve gives them, the requests of the notifications, a tab whose caller nerve listed acme
// and beta, and the fetch that loads a workspace's count into the store.

import type { NotificationUnreadCount, Workspace } from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { fakeRoot, withoutImageObservation } from "@/store/fake-root";
import type { INotificationStore } from "@/store/notification/notification.store";
import { NotificationStore } from "@/store/notification/notification.store";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

export const acme = workspaceOf("acme");
export const beta = workspaceOf("beta");

/** The caller's unread notifications in a workspace as nerve counts them: all is every one but the mentions. */
export const countOf = (all: number, mentions: number): NotificationUnreadCount => ({ all, mentions });

/** The requests of the notifications: a workspace's unread counts, by its slug. */
export const notificationRequests = {
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
