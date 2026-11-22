/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { cn } from "@nerve/utils";
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
import { useLeavingFocus } from "./use-leaving-focus";

/** The list's own element, which the focus has once no card is left: its ring inside its edge, as the cards' have. */
const REGION = "focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong";

/**
 * The list of the notifications page's tab (P4 spec 2.6, R7, R21): the route's workspace's notifications its filters
 * select, as nerve gives them a page at a time (useNotificationList). Until nerve answers, a wait; why nerve could not
 * give it, with a retry (never a wait that does not end); the tab's empty state; else a card for each, and "Load more"
 * while nerve gives a cursor, even over a page of no rows (its rows archived or snoozed away, re-93 m6), out of use
 * while it reads the next page, a toast saying why nerve could not. As a card that holds the focus leaves it (its own
 * archive or snooze), the focus goes to the next card, else the previous, else the list itself, focusable from script
 * whatever it shows (useLeavingFocus). No region is named "Notifications": that is the toasts' (the e2e's toast).
 */
export const NotificationList = observer(function NotificationList() {
  const { t } = useTranslation();
  const { filters } = useNotificationPageState();
  const { workspace, page, read } = useNotificationList(filters);
  const { fetchMore } = useNotifications();
  const refusalToast = useRefusalToast();
  const [loadingMore, setLoadingMore] = useState(false);
  const { region, focusIn, focusOut } = useLeavingFocus(page?.data.map((notification) => notification.id) ?? []);

  if (!workspace) return null;
  if (read.error !== undefined) return <PlanningReadFailure error={read.error} onRetry={() => void read.mutate()} />;
  if (page === undefined) return <NotificationsLoader />;
  if (page.data.length === 0 && page.next_cursor === null)
    return (
      <div ref={region} tabIndex={-1} className={cn("flex size-full items-center justify-center", REGION)}>
        <NotificationEmptyState tab={filters.tab} />
      </div>
    );

  const loadMore = async () => {
    setLoadingMore(true);
    await followInSession(() => fetchMore(workspace, filters), { failed: refusalToast });
    setLoadingMore(false);
  };

  return (
    <div ref={region} tabIndex={-1} className={REGION}>
      <ul>
        {page.data.map((notification) => (
          <li
            key={notification.id}
            data-notification-id={notification.id}
            onFocus={() => focusIn(notification.id)}
            onBlur={(event) => focusOut(event.target, event.relatedTarget)}
          >
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
