/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// components
import { PlanningReadFailure } from "@/components/planning/read-failure";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { NotificationItem } from "../card/item";
import { useNotificationPageState } from "../page-state";
import { useNotificationList } from "../use-notification-list";
import { NotificationEmptyState } from "./empty-state";
import { NotificationsLoader } from "./loader";

/**
 * The list of the notifications page's tab (P4 spec 2.6, R7, R21): the route's workspace's notifications its filters
 * select, as nerve gives them a page at a time (useNotificationList). Until nerve answers, a wait; why nerve could not
 * give it, with a retry (never a wait that does not end); the tab's empty state; else a card for each, and "Load more"
 * while nerve gives a cursor, even over a page of no rows (its rows archived or snoozed away, re-93 m6), out of use
 * while it reads the next page, a toast saying why nerve could not. No region is named "Notifications": that is the
 * toasts' (the e2e's toast).
 */
export const NotificationList = observer(function NotificationList() {
  const { t } = useTranslation();
  const { filters } = useNotificationPageState();
  const { workspace, page, read } = useNotificationList(filters);
  const { fetchMore } = useNotifications();
  const refusalToast = useRefusalToast();
  const [loadingMore, setLoadingMore] = useState(false);

  if (!workspace) return null;
  if (read.error !== undefined) return <PlanningReadFailure error={read.error} onRetry={() => void read.mutate()} />;
  if (page === undefined) return <NotificationsLoader />;
  if (page.data.length === 0 && page.next_cursor === null)
    return (
      <div className="flex size-full items-center justify-center">
        <NotificationEmptyState tab={filters.tab} />
      </div>
    );

  const loadMore = async () => {
    setLoadingMore(true);
    await followInSession(() => fetchMore(workspace, filters), { failed: refusalToast });
    setLoadingMore(false);
  };

  return (
    <div>
      <ul>
        {page.data.map((notification) => (
          <li key={notification.id}>
            <NotificationItem notification={notification} />
          </li>
        ))}
      </ul>
      {page.next_cursor !== null && (
        <div className="flex justify-center py-4">
          <Button variant="ghost" size="sm" disabled={loadingMore} onClick={() => void loadMore()}>
            {loadingMore ? t("common.loading") : t("common.load_more")}
          </Button>
        </div>
      )}
    </div>
  );
});
