/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { FilterOutline } from "@makeplane/propel/icons";
// nerve imports
import { NOTIFICATION_TYPE_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { PopoverMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationMenuToggle } from "./menu-toggle";

/**
 * The type filter of the notifications page's all tab (P4 spec 2.6, R10, R16, re-93 I3): the senders its list shows,
 * every one while none is chosen. A menu (PopoverMenu) whose button is an icon, its name for a screen reader and a dot
 * while some are chosen, with no button inside it (W7); its items toggle the page's choice (NotificationMenuToggle:
 * Tab between them, Enter and Space once each); Escape closes it and gives the focus back to its button. Not there in
 * the mentions tab, where nerve takes no type (the contract's T2): the all tab's choice stays for its return.
 */
export const NotificationTypeFilterMenu = observer(function NotificationTypeFilterMenu() {
  const { t } = useTranslation();
  const { filters, toggleType } = useNotificationPageState();
  if (filters.tab !== "all") return null;
  return (
    <PopoverMenu
      buttonClassName={cn(getIconButtonStyling("ghost", "base"), "relative")}
      button={
        <>
          <FilterOutline className="size-4" aria-hidden="true" />
          <span className="sr-only">{t("notification.options.filters")}</span>
          {filters.types.length > 0 && (
            <span
              aria-hidden="true"
              className="absolute top-0.5 right-0.5 h-1.5 w-1.5 rounded-full bg-accent-primary"
            />
          )}
        </>
      }
      panelClassName="p-2 rounded-md border border-subtle bg-surface-1"
      data={[...NOTIFICATION_TYPE_OPTIONS]}
      keyExtractor={(option) => option.value}
      render={(option) => (
        <NotificationMenuToggle
          label={t(option.i18n_label)}
          pressed={filters.types.includes(option.value)}
          onToggle={() => toggleType(option.value)}
        />
      )}
    />
  );
});
