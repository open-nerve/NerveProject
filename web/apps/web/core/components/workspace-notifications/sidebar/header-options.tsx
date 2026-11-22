/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { CheckDoneOutline, RefreshOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Spinner } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// store
import { filterKey } from "@/store/notification/filters";
// local imports
import { useNotificationPageState } from "../page-state";
import { useNotificationList } from "../use-notification-list";
import { NotificationTypeFilterMenu } from "./type-filter-menu";
import { NotificationViewMenu } from "./view-menu";

/**
 * The head's buttons of the notifications page (P4 spec 2.6, R7, R9, R10), for the route's workspace. Mark all read
 * marks the unread notifications the page's filters select (the store's markAllRead, its body made of them: the all
 * tab's senders only), out of use and turning while nerve marks; says how many nerve marked, or why it refused; either
 * way the list's first page is read again (as Refresh reads it, nerve marking in batches) and the store reads the count
 * again. Refresh reads the first page the filters select, in place of every page held (R7), and the count again, out of
 * use and turning meanwhile, saying why nerve could not. Both read the list the page shows: the store's read is given
 * to the list's read by its key (useNotificationList, which the head shares with the list, one read as the page opens),
 * so that the list shows the page once it is read, a failure of its read gone, and SWR reads nothing more; mark all
 * read's, made once nerve has answered, only while the page still shows the filters it marked, else the store reads
 * them alone (the list's read is then another key's: whole-branch m2). What follows nerve's answer is done only in the
 * session it was sent in (followInSession). Then the type filter (all tab alone) and the view menu.
 */
export const NotificationHeaderOptions = observer(function NotificationHeaderOptions() {
  const { t } = useTranslation();
  const { filters } = useNotificationPageState();
  const { workspace, read } = useNotificationList(filters);
  const { markAllRead, fetchList, fetchUnreadCount } = useNotifications();
  const refusalToast = useRefusalToast();
  const [marking, setMarking] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  // the filters the page shows now, which a late answer compares with the ones it was sent for
  const shown = useRef(filters);
  useEffect(() => {
    shown.current = filters;
  }, [filters]);

  if (!workspace) return null;

  /**
   * The list's first page read again, given to its key while the page shows these filters still (SWR's mutate is the
   * key's the page shows now), as Refresh does; a failure leaves the list as it was.
   */
  const readAgain = () => {
    const again = fetchList(workspace, filters);
    const shows = filterKey(shown.current) === filterKey(filters);
    void Promise.allSettled([shows ? read.mutate(again, { revalidate: false }) : again]);
  };

  const markAll = async () => {
    setMarking(true);
    await followInSession(() => markAllRead(workspace, filters), {
      done: ({ marked }) => {
        readAgain();
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("notification.toasts.all_read", { count: marked }),
        });
      },
      failed: (error) => {
        readAgain();
        refusalToast(error);
      },
    });
    setMarking(false);
  };

  const refresh = async () => {
    setRefreshing(true);
    const list = () => read.mutate(fetchList(workspace, filters), { revalidate: false });
    await followInSession(() => Promise.all([list(), fetchUnreadCount(workspace)]), { failed: refusalToast });
    setRefreshing(false);
  };

  const markAllLabel = t("notification.options.mark_all_as_read");
  const refreshLabel = t("notification.options.refresh");
  return (
    <div className="relative flex items-center justify-center gap-1">
      <Tooltip label={markAllLabel} side="bottom">
        <IconButton
          size="base"
          variant="ghost"
          icon={marking ? Spinner : CheckDoneOutline}
          aria-label={markAllLabel}
          disabled={marking}
          onClick={() => void markAll()}
        />
      </Tooltip>
      <Tooltip label={refreshLabel} side="bottom">
        <IconButton
          size="base"
          variant="ghost"
          icon={RefreshOutline}
          iconClassName={cn({ "animate-spin": refreshing })}
          aria-label={refreshLabel}
          disabled={refreshing}
          onClick={() => void refresh()}
        />
      </Tooltip>
      <NotificationTypeFilterMenu />
      <NotificationViewMenu />
    </div>
  );
});
