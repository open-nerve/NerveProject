/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { Avatar } from "@makeplane/propel/components/avatar";
import { ClockOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Notification } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { calculateTimeAgo, cn, renderFormattedDate } from "@nerve/utils";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useActorName } from "@/hooks/use-actor-name";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { followInSession } from "@/lib/in-session";
// store
import { snoozedNow } from "@/store/notification/filters";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationActions } from "./actions";
import { NotificationSentence } from "./sentence";
import { timeLabel } from "./snooze-time";
import { useNotificationRefusal } from "./use-notification-refusal";

type Props = {
  /** The notification the card shows, as nerve last gave it. */
  notification: Notification;
};

/**
 * A card of the notifications page's list (P4 spec 2.6, R16, R23): its main button selects its notification, which
 * opens the preview at once, and, for an unread one, queues marking it read (the store's update, one change at a time),
 * which the preview never waits for: nerve's refusal of that is said in a toast, the preview open still, but for
 * notification.not_found, after which the notification has left every list and its selection is cleared
 * (useNotificationRefusal). Beside the main button, never in it, its actions (actions.tsx). The main button
 * shows who made the change, his picture for a member (useActorName), and says what (sentence.tsx); a comment's excerpt,
 * as the plain text nerve keeps of it, two lines at most (no editor, no picture, no mention's name: M5-M7-03); the work
 * item, as it was when the notification was sent; and when: till when it is snoozed, while it is (snoozedNow: a snooze
 * over is an ordinary notification's), its time of day in the page's language (timeLabel), else how long ago. An
 * unread one's dot says so to a screen reader too. Nothing in it is a link, an attachment's either (M5-M7-02). What
 * follows nerve's answer is done only in the session the change was sent in (followInSession).
 */
export const NotificationItem = observer(function NotificationItem({ notification }: Props) {
  const { t, language } = useTranslation();
  const workspace = useRouteWorkspace();
  const { selected, select } = useNotificationPageState();
  const { update } = useNotifications();
  const refused = useNotificationRefusal(notification.id);
  const actor = useActorName(notification.triggered_by_id);
  // derived values
  const { id, issue_id: issueId, project_id: projectId, triggered_by_id: actorId } = notification;
  const { excerpt } = notification.data.change;
  const initial = actor.name.charAt(0).toUpperCase();
  const issue = notification.data.issue;
  const unread = notification.read_at === null;
  const till = snoozedNow(notification, new Date()) ? notification.snoozed_till : null;

  const open = () => {
    select({ notificationId: id, issueId, projectId });
    if (!unread || !workspace) return;
    void followInSession(() => update(workspace, id, { read: true }), { failed: refused });
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
        className="flex w-full gap-3 px-6 py-4 text-left focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong"
        onClick={open}
      >
        {unread && (
          <span className="absolute top-1/2 left-2 h-1.5 w-1.5 rounded-full bg-accent-primary">
            <span className="sr-only">{t("notification.card.unread")}</span>
          </span>
        )}
        {actor.member && actorId !== null && workspace ? (
          <ProfileAvatar
            size="lg"
            fallback={initial}
            source={{ kind: "member", workspaceID: workspace.id, memberID: actorId }}
          />
        ) : (
          <Avatar size="lg" fallback={initial} />
        )}
        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <NotificationSentence notification={notification} />
          {excerpt !== null && (
            <span className="line-clamp-2 text-caption-sm-regular break-words text-secondary">{excerpt}</span>
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
                  time: timeLabel(new Date(till), language),
                })}
              </span>
            ) : (
              <span className="flex-shrink-0 text-tertiary">{calculateTimeAgo(notification.created_at)}</span>
            )}
          </span>
        </span>
      </button>
      <NotificationActions notification={notification} />
    </div>
  );
});
