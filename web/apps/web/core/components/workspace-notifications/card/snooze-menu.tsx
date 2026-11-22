/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { ClockOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Notification } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { CustomMenu } from "@nerve/ui";
// store
import { snoozedNow } from "@/store/notification/filters";
// local imports
import { SNOOZE_PRESETS, presetMoment } from "./snooze-time";

type Props = {
  /** The notification of the card, as nerve last gave it. */
  notification: Notification;
  /** The menu's name, its button's: the action and its row. */
  label: string;
  /** Whether the menu is out of use, while a change of the card is out. */
  disabled: boolean;
  /** Sends the snooze: till, a moment, or null to end it; done, the toast's key that says nerve made it. */
  onSnooze: (till: string | null, done: string) => void;
};

/**
 * A card's snooze menu (P4 spec 2.6, R15, R16): a menu (CustomMenu) whose button, an icon named by label, opens it by
 * the pointer or the keyboard, the arrow keys moving among its items, Enter choosing one, Escape closing it and giving
 * the focus back to its button. A snoozed one (snoozedNow) first ends its snooze, which is snoozed_till null (the
 * contract's T1: a field left out changes nothing); then the presets, each from the moment it is chosen
 * (presetMoment).
 */
export function NotificationSnoozeMenu({ notification, label, disabled, onSnooze }: Props) {
  const { t } = useTranslation();
  return (
    <CustomMenu
      customButton={<ClockOutline className="size-3.5" />}
      customButtonClassName={getIconButtonStyling("ghost", "sm")}
      ariaLabel={label}
      disabled={disabled}
      placement="bottom-end"
    >
      {snoozedNow(notification, new Date()) && (
        <CustomMenu.MenuItem onClick={() => onSnooze(null, "notification.toasts.unsnoozed")}>
          {t("notification.options.mark_unsnooze")}
        </CustomMenu.MenuItem>
      )}
      {SNOOZE_PRESETS.map((preset) => (
        <CustomMenu.MenuItem
          key={preset.days}
          onClick={() => onSnooze(presetMoment(preset.days, new Date()), "notification.toasts.snoozed")}
        >
          {t(preset.i18n_label)}
        </CustomMenu.MenuItem>
      ))}
    </CustomMenu>
  );
}
