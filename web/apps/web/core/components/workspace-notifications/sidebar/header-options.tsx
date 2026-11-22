/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState } from "react";
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
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationTypeFilterMenu } from "./type-filter-menu";
import { NotificationViewMenu } from "./view-menu";

/**
 * The head's buttons of the notifications page (P4 spec 2.6, R7, R9, R10), for the route's workspace. Mark all read
 * marks the unread notifications the page's filters select (the store's markAllRead, its body made of them: the all
 * tab's senders only), out of use and turning while nerve marks; says how many nerve marked, or why it refused; either
 * way the store reads the list's first page and the count again, nerve marking in batches. Refresh reads the first page
 * the filters select, in place of every page held (R7), and the count again, out of use and turning meanwhile, saying
 * why nerve could not. What follows nerve's answer is done only in the session it was sent in (followInSession). Then
 * the type filter (all tab alone) and the view menu.
 */
export const NotificationHeaderOptions = observer(function NotificationHeaderOptions() {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const { filters } = useNotificationPageState();
  const { markAllRead, fetchList, fetchUnreadCount } = useNotifications();
  const refusalToast = useRefusalToast();
  const [marking, setMarking] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  if (!workspace) return null;

  const markAll = async () => {
    setMarking(true);
    await followInSession(() => markAllRead(workspace, filters), {
      done: ({ marked }) =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("notification.toasts.all_read", { count: marked }),
        }),
      failed: refusalToast,
    });
    setMarking(false);
  };

  const refresh = async () => {
    setRefreshing(true);
    await followInSession(() => Promise.all([fetchList(workspace, filters), fetchUnreadCount(workspace)]), {
      failed: refusalToast,
    });
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
