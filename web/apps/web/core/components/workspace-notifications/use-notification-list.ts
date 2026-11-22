/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NotificationPage, Workspace } from "@nerve/api-client";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// store
import type { NotificationFilters } from "@/store/notification/filters";
import { filterKey, filtersOf } from "@/store/notification/filters";

/** The notifications page's list: its route's workspace, the pages the store holds of it, and SWR's read of it. */
type NotificationList = {
  workspace: Workspace | null;
  page: NotificationPage | undefined;
  read: ReturnType<typeof useSessionSWR<NotificationPage | undefined>>;
};

/**
 * The read of the route's workspace's notifications the filters select (P4 spec 2.4), by the session's SWR keyed by
 * the workspace's id and slug and the filters' key (filterKey), so that a change of filters reads the list they name;
 * the fetcher is given the key's strings alone and reads the filters they name (filtersOf), never the filters of the
 * render that made it (use-session-swr.ts: what it fetches is what the key names). For every role of the workspace;
 * none for a route that names no workspace of the caller's. SWR reads it as the page mounts and as the key changes,
 * and again as the browser reconnects, which is a refresh: the first page in place of every page held (R7). Gives the
 * workspace, the list as the store holds it (getList: the pages read so far as one, undefined until read), and SWR's
 * read, whose error is why nerve could not read it.
 */
export function useNotificationList(filters: NotificationFilters): NotificationList {
  const workspace = useRouteWorkspace();
  const { fetchList, getList } = useNotifications();
  const read = useSessionSWR(
    workspace && ["NOTIFICATIONS", workspace.id, workspace.slug, filterKey(filters)],
    (id, slug, key) => fetchList({ id, slug }, filtersOf(key))
  );
  return { workspace, page: workspace ? getList(workspace.slug, filters) : undefined, read };
}
