/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Link } from "react-router";
import type { ProjectNavigation } from "@nerve/api-client";
import { DefaultTabOutline, MoreHorizontalOutline, PinOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Menu } from "@nerve/propel/menu";
import { cn } from "@nerve/utils";
// local imports
import type { TNavigationItem } from "./tab-navigation-root";
import type { TTabChanges } from "./use-tab-preferences";

type Props = {
  overflowItems: TNavigationItem[];
  isActive: (item: TNavigationItem) => boolean;
  navigation: ProjectNavigation;
  /** The tab bar's changes; none until the caller's tab bar is fetched, and the menu then offers none. */
  changes: TTabChanges | undefined;
};

// an action of a tab's row: its icon shows while the pointer is over the row or the arrow keys are on an item of it,
// and always for the tab the project opens on (shown); never hidden from the arrow keys, which reach each action
const ACTION_CLASS =
  "w-auto shrink-0 p-1 text-tertiary opacity-0 transition-colors group-hover/menu-item:opacity-100 group-has-data-highlighted/menu-item:opacity-100 hover:text-primary data-highlighted:text-primary";

/**
 * Overflow menu for tab navigation items: those that don't fit in the visible area, and those the caller hid. Each tab
 * is an item of the menu's, a link to its page; once his tab bar is fetched (changes), each tab's row has items for
 * its actions too, each named by its tab: "Show" for a tab he hid, and "Set as default" (or "Clear default") for every
 * tab; until then, no action. The arrow keys reach each item, and Enter does what a click does (C11-2).
 */
export function TabNavigationOverflowMenu({ overflowItems, isActive, navigation, changes }: Props) {
  const { t } = useTranslation();

  return (
    <Menu
      ellipsis
      ariaLabel={t("tab_navigation.more_tabs")}
      buttonClassName="!p-1.5"
      optionsClassName="min-w-[200px] space-y-1"
      customButton={
        <div className="flex items-center justify-center rounded-md p-1 transition-colors hover:bg-layer-1">
          <MoreHorizontalOutline className="h-4 w-4 text-secondary" />
        </div>
      }
    >
      {overflowItems.map((item) => {
        const itemIsActive = isActive(item);
        // isHidden = true only for user-hidden items (not space-constrained overflow)
        const isHidden = navigation.hide_in_more_menu.includes(item.key);
        const isDefault = item.key === navigation.default_tab;
        const tab = t(item.i18n_key);

        return (
          <div
            key={`${item.key}-overflow-${itemIsActive ? "active" : "inactive"}`}
            className="group/menu-item flex w-full items-center justify-between"
          >
            <Menu.MenuItem render={<Link to={item.href} />} className="min-w-0 flex-1 p-1">
              <span className="text-11">{tab}</span>
            </Menu.MenuItem>
            {changes && isHidden && (
              <Menu.MenuItem
                onClick={() => changes.show(item.key)}
                className={ACTION_CLASS}
                title={t("tab_navigation.show_tab", { tab })}
              >
                <PinOutline className="size-3" />
              </Menu.MenuItem>
            )}
            {changes && (
              <Menu.MenuItem
                onClick={() => changes.toggleDefault(item.key)}
                className={cn(ACTION_CLASS, { "opacity-100": isDefault })}
                title={t(isDefault ? "tab_navigation.clear_tab_default" : "tab_navigation.set_tab_as_default", { tab })}
              >
                <DefaultTabOutline className="size-3" />
              </Menu.MenuItem>
            )}
          </div>
        );
      })}
    </Menu>
  );
}
