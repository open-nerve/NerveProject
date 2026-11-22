/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { ArchiveOutline, CheckDoneOutline, MailOutline, RestoreOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Notification, NotificationUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useNotifications } from "@/hooks/store/use-notifications";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { followInSession } from "@/lib/in-session";
// store
import { viewOffers } from "@/store/notification/filters";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationSnoozeMenu } from "./snooze-menu";
import { useNotificationRefusal } from "./use-notification-refusal";

type Props = {
  /** The notification of the card, as nerve last gave it. */
  notification: Notification;
};

/**
 * A card's actions (P4 spec 2.6, R16, R25), beside its main button: mark read or unread; archive or unarchive, unless
 * the snoozed are listed; the snooze menu (snooze-menu.tsx), unless the archived are (viewOffers: the page never makes
 * a notification both archived and snoozed, which nerve lists in no view). Each is a button named by its row, its work
 * item's reference and name, in the Tab order after the main button; they show as the card is hovered or holds the
 * focus, never hidden from the keyboard until then. Each sends its change through the store (update: one change at a
 * time, nerve's answer written into every list held), all of them out of use meanwhile; says it once nerve has made
 * it, or why not (useNotificationRefusal), never both. What follows nerve's answer is done only in the session the
 * change was sent in (followInSession).
 */
export const NotificationActions = observer(function NotificationActions({ notification }: Props) {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const { filters } = useNotificationPageState();
  const { update } = useNotifications();
  const refused = useNotificationRefusal(notification.id);
  const [sending, setSending] = useState(false);

  if (!workspace) return null;

  const offers = viewOffers(filters);
  const { issue } = notification.data;
  const row = `${issue.identifier}-${issue.sequence_id} ${issue.name}`;
  const unread = notification.read_at === null;
  const archived = notification.archived_at !== null;

  /** Sends the change; once nerve has made it, says so in the words of done, a toast's key. */
  const send = async (patch: NotificationUpdate, done: string) => {
    setSending(true);
    await followInSession(() => update(workspace, notification.id, patch), {
      done: () => setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t(done) }),
      failed: refused,
    });
    setSending(false);
  };

  return (
    <div className="absolute top-3 right-4 flex items-center gap-1 rounded-md bg-surface-1 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
      <Tooltip label={unread ? t("notification.options.mark_read") : t("notification.options.mark_unread")}>
        <IconButton
          size="sm"
          variant="ghost"
          icon={unread ? CheckDoneOutline : MailOutline}
          aria-label={
            unread
              ? t("notification.card.actions.mark_read", { ref: row })
              : t("notification.card.actions.mark_unread", { ref: row })
          }
          disabled={sending}
          onClick={() =>
            void send({ read: unread }, unread ? "notification.toasts.read" : "notification.toasts.unread")
          }
        />
      </Tooltip>
      {offers.archive && (
        <Tooltip label={archived ? t("notification.options.mark_unarchive") : t("notification.options.mark_archive")}>
          <IconButton
            size="sm"
            variant="ghost"
            icon={archived ? RestoreOutline : ArchiveOutline}
            aria-label={
              archived
                ? t("notification.card.actions.unarchive", { ref: row })
                : t("notification.card.actions.archive", { ref: row })
            }
            disabled={sending}
            onClick={() =>
              void send(
                { archived: !archived },
                archived ? "notification.toasts.unarchived" : "notification.toasts.archived"
              )
            }
          />
        </Tooltip>
      )}
      {offers.snooze && (
        <NotificationSnoozeMenu
          notification={notification}
          label={t("notification.card.actions.snooze", { ref: row })}
          disabled={sending}
          onSnooze={(till, done) => void send({ snoozed_till: till }, done)}
        />
      )}
    </div>
  );
});
