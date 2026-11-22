/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
// nerve imports
import type { Notification } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedDate } from "@nerve/utils";
// hooks
import { useActorName } from "@/hooks/use-actor-name";
// local imports
import type { ChangeFormat } from "./describe-change";
import { describeChange } from "./describe-change";

type Props = {
  /** The notification whose change the sentence says. */
  notification: Notification;
};

/**
 * A notification card's sentence (P4 spec 2.6, R11, R22): who made the change (useActorName) and what, as
 * describeChange says it, in the page's language. A day nerve gives is a calendar day ("2026-10-17"), shown as that day
 * (renderFormattedDate reads it through getDate), never as a moment, which a zone west of UTC shows as the day before
 * (re-93 m15); a priority by its text.
 */
export const NotificationSentence = observer(function NotificationSentence({ notification }: Props) {
  const { t } = useTranslation();
  const actor = useActorName(notification.triggered_by_id);
  const format: ChangeFormat = {
    date: (day) => renderFormattedDate(day) ?? day,
    priority: (priority) => t(priority),
  };
  const { key, params } = describeChange(notification.data.change, format);
  return (
    <span className="line-clamp-2 text-body-xs-regular break-words text-tertiary">
      <span className="font-medium text-primary">{actor.name}</span> {t(key, params)}
    </span>
  );
});
