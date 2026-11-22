/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NotificationUnreadCount } from "@nerve/api-client";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The one fetch of the caller's unread notifications in the route's workspace (P4 spec 2.4, R5), which the top
 * navigation of every page of a workspace and the notifications page's tabs share: keyed by the workspace's id, from
 * its slug's address, for every role of it (an admin's, a member's, a guest's); none for a route that names no
 * workspace of his. SWR fetches it as a page mounts and as the workspace changes, never as the tab regains focus (R6):
 * the writes that change it fetch it again through the store. Gives the counts the store holds, undefined until
 * fetched.
 */
export function useUnreadNotificationCount(): NotificationUnreadCount | undefined {
  const workspace = useRouteWorkspace();
  const { fetchUnreadCount, getUnreadCount } = useNotifications();
  useSessionSWR(workspace && ["NOTIFICATION_UNREAD_COUNT", workspace.id, workspace.slug], (id, slug) =>
    fetchUnreadCount({ id, slug })
  );
  return workspace ? getUnreadCount(workspace.slug) : undefined;
}
