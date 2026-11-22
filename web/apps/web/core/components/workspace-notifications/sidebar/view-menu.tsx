/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { MoreVerticalOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { PopoverMenu } from "@nerve/ui";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationMenuToggle } from "./menu-toggle";

/**
 * The view menu of the notifications page (P4 spec 2.5, 2.6, R16, R25, re-93 I3): the unread alone, the archived alone,
 * the snoozed alone, in either tab. The archived and the snoozed exclude each other (the page's state turns one off as
 * the other goes on), since nerve takes both as equalities and the page never asks for both. A menu (PopoverMenu)
 * whose button is an icon and its name for a screen reader, with no button inside it (W7); its items toggle the views
 * (NotificationMenuToggle: Tab between them, Enter and Space once each); Escape closes it and gives the focus back to
 * its button.
 */
export const NotificationViewMenu = observer(function NotificationViewMenu() {
  const { t } = useTranslation();
  const { filters, toggleUnread, toggleArchived, toggleSnoozed } = useNotificationPageState();
  // the views, each a toggle of the page's filters
  const views = [
    { key: "unread", i18n_label: "notification.options.show_unread", pressed: filters.unread, toggle: toggleUnread },
    {
      key: "archived",
      i18n_label: "notification.options.show_archived",
      pressed: filters.archived,
      toggle: toggleArchived,
    },
    {
      key: "snoozed",
      i18n_label: "notification.options.show_snoozed",
      pressed: filters.snoozed,
      toggle: toggleSnoozed,
    },
  ];
  return (
    <PopoverMenu
      buttonClassName={getIconButtonStyling("ghost", "base")}
      button={
        <>
          <MoreVerticalOutline className="size-4" aria-hidden="true" />
          <span className="sr-only">{t("common.display")}</span>
        </>
      }
      panelClassName="p-2 rounded-md border border-subtle bg-surface-1"
      data={views}
      keyExtractor={(view) => view.key}
      render={(view) => (
        <NotificationMenuToggle label={t(view.i18n_label)} pressed={view.pressed} onToggle={view.toggle} />
      )}
    />
  );
});
