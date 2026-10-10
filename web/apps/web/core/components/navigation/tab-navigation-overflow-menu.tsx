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
import { Tooltip } from "@makeplane/propel/components/tooltip";
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

/**
 * Overflow menu for tab navigation items
 * Displays items that don't fit in the visible area, with action icons
 * Shows "Eye" icon for user-hidden items, "Set as default" icon for all items
 */
export function TabNavigationOverflowMenu({ overflowItems, isActive, navigation, changes }: Props) {
  const { t } = useTranslation();

  return (
    <Menu
      ellipsis
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

        return (
          <Menu.MenuItem key={`${item.key}-overflow-${itemIsActive ? "active" : "inactive"}`} className="w-full p-0">
            <div className="group/menu-item flex w-full items-center justify-between">
              <Link to={item.href} className="w-full min-w-0 flex-1 p-1">
                <span className="text-11">{t(item.i18n_key)}</span>
              </Link>
              <div className="flex items-center">
                {/* Show Eye icon ONLY for user-hidden items */}
                {changes && isHidden && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      e.preventDefault();
                      changes.show(item.key);
                    }}
                    className="invisible rounded-sm p-1 text-tertiary transition-colors group-hover/menu-item:visible hover:text-primary"
                    title="Show"
                  >
                    <PinOutline className="size-3" />
                  </button>
                )}
                {changes && (
                  <Tooltip label={isDefault ? "Clear default" : "Set as default"}>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                        changes.toggleDefault(item.key);
                      }}
                      className={cn(
                        "invisible rounded-sm p-1 text-tertiary transition-colors group-hover/menu-item:visible hover:text-primary",
                        {
                          visible: isDefault,
                        }
                      )}
                      title={isDefault ? "Clear default" : "Set as default"}
                    >
                      <DefaultTabOutline className="size-3" />
                    </button>
                  </Tooltip>
                )}
              </div>
            </div>
          </Menu.MenuItem>
        );
      })}
    </Menu>
  );
}
