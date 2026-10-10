/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import type { ProjectNavigation } from "@nerve/api-client";
import { DefaultTabOutline, UnpinOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { ContextMenu } from "@nerve/propel/context-menu";
import { TabNavigationItem } from "@nerve/propel/tab-navigation";
// local imports
import type { TNavigationItem } from "./tab-navigation-root";
import type { TTabChanges } from "./use-tab-preferences";

export type TTabNavigationVisibleItemProps = {
  item: TNavigationItem;
  isActive: boolean;
  navigation: ProjectNavigation;
  /** The tab bar's changes; none until the caller's tab bar is fetched, and the tab then has no menu. */
  changes: TTabChanges | undefined;
  itemRef?: (el: HTMLDivElement | null) => void;
};

/**
 * Individual visible tab navigation item with context menu
 * Handles right-click actions for setting default and hiding tabs
 */
export function TabNavigationVisibleItem({
  item,
  isActive,
  navigation,
  changes,
  itemRef,
}: TTabNavigationVisibleItemProps) {
  const { t } = useTranslation();
  const isDefault = item.key === navigation.default_tab;
  const link = (
    <Link key={`${item.key}-${isActive ? "active" : "inactive"}`} to={item.href}>
      <TabNavigationItem isActive={isActive}>
        <span>{t(item.i18n_key)}</span>
      </TabNavigationItem>
    </Link>
  );

  return (
    <div className="relative flex h-full items-center transition-all duration-300">
      {isActive && (
        <span className="absolute bottom-0 left-1/2 h-0.5 w-[80%] -translate-x-1/2 rounded-t-md bg-(--text-color-icon-primary) transition-all duration-300" />
      )}
      <div key={`${item.key}-measure`} ref={itemRef}>
        {changes ? (
          <ContextMenu>
            <ContextMenu.Trigger>{link}</ContextMenu.Trigger>
            <ContextMenu.Portal>
              <ContextMenu.Content positionerClassName="z-30">
                <ContextMenu.Item
                  onClick={(e) => {
                    e.stopPropagation();
                    changes.toggleDefault(item.key);
                  }}
                  className="flex cursor-pointer items-center gap-2 text-secondary transition-colors"
                >
                  <DefaultTabOutline className="size-3 shrink-0" />
                  <span className="text-11">{isDefault ? "Clear default" : "Set as default"}</span>
                </ContextMenu.Item>
                <ContextMenu.Item
                  onClick={(e) => {
                    e.stopPropagation();
                    changes.hide(item.key);
                  }}
                  className="flex cursor-pointer items-center gap-2 text-secondary transition-colors"
                >
                  <UnpinOutline className="size-3 shrink-0" />
                  <span className="text-11">Hide in more menu</span>
                </ContextMenu.Item>
              </ContextMenu.Content>
            </ContextMenu.Portal>
          </ContextMenu>
        ) : (
          link
        )}
      </div>
    </div>
  );
}
