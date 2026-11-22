/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { ClockOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Notification } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { calculateTimeAgo, cn, renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { problemCode } from "@/lib/api-error";
import { followInSession } from "@/lib/in-session";
// store
import { snoozedNow } from "@/store/notification/filters";
// local imports
import { useNotificationPageState } from "../page-state";

type Props = {
  /** The notification the card shows, as nerve last gave it. */
  notification: Notification;
};

/**
 * A card of the notifications page's list (P4 spec 2.6, R16, R23): its one button selects its notification, which
 * opens the preview at once, and, for an unread one, queues marking it read (the store's update, one change at a time),
 * which the preview never waits for: nerve's refusal of that is said in a toast, the preview open still, but for
 * notification.not_found, after which the notification has left every list and its selection is cleared. The button
 * says the work item, as it was when the notification was sent, and when: till when it is snoozed, while it is
 * (snoozedNow: a snooze over is an ordinary notification's), else how long ago; an unread one's dot says so to a screen
 * reader too. What follows nerve's answer is done only in the session the change was sent in (followInSession).
 */
export const NotificationItem = observer(function NotificationItem({ notification }: Props) {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const { selected, select, clearSelection } = useNotificationPageState();
  const { update } = useNotifications();
  const refusalToast = useRefusalToast();
  // derived values
  const { id, issue_id: issueId, project_id: projectId } = notification;
  const issue = notification.data.issue;
  const unread = notification.read_at === null;
  const till = snoozedNow(notification, new Date()) ? notification.snoozed_till : null;

  const open = () => {
    select({ notificationId: id, issueId, projectId });
    if (!unread || !workspace) return;
    void followInSession(() => update(workspace, id, { read: true }), {
      failed: (error) => {
        refusalToast(error);
        if (problemCode(error) === "notification.not_found") clearSelection(id);
      },
    });
  };

  return (
    <div
      className={cn("group relative border-b border-subtle transition-all", {
        "bg-layer-1/30": selected?.notificationId === id,
        "bg-accent-primary/5": unread,
      })}
    >
      <button
        type="button"
        className="flex w-full flex-col gap-1 px-6 py-4 text-left focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong"
        onClick={open}
      >
        {unread && (
          <span className="absolute top-1/2 left-2 h-1.5 w-1.5 rounded-full bg-accent-primary">
            <span className="sr-only">{t("notification.card.unread")}</span>
          </span>
        )}
        <span className="flex w-full items-center gap-3 text-caption-sm-regular text-secondary">
          <span className="line-clamp-1 w-full break-words">
            {`${issue.identifier}-${issue.sequence_id} ${issue.name}`}
          </span>
          {till !== null ? (
            <span className="flex flex-shrink-0 items-center gap-x-1 text-tertiary">
              <ClockOutline className="h-4 w-4" />
              {t("notification.card.till", {
                date: renderFormattedDate(new Date(till)),
                time: renderFormattedTime(new Date(till), "12-hour"),
              })}
            </span>
          ) : (
            <span className="flex-shrink-0 text-tertiary">{calculateTimeAgo(notification.created_at)}</span>
          )}
        </span>
      </button>
    </div>
  );
});
