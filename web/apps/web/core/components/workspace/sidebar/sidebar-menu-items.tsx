/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ChevronRightOutline } from "@makeplane/propel/icons";
import { Transition } from "@headlessui/react";
// nerve imports
import {
  WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,
  WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS,
} from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// hooks
import useLocalStorage from "@/hooks/use-local-storage";
import { SidebarItemBase } from "./sidebar-item";

export const SidebarMenuItems = observer(function SidebarMenuItems() {
  // routers
  const { setValue: toggleWorkspaceMenu, storedValue: isWorkspaceMenuOpen } = useLocalStorage<boolean>(
    "is_workspace_menu_open",
    true
  );

  // translation
  const { t } = useTranslation();

  const toggleListDisclosure = (isOpen: boolean) => {
    toggleWorkspaceMenu(isOpen);
  };

  return (
    <>
      <div className="flex flex-col gap-0.5">
        {WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS.map((item) => (
          <SidebarItemBase key={item.key} item={item} />
        ))}
      </div>
      <div className="flex flex-col">
        <div className="group flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-placeholder hover:bg-layer-transparent-hover">
          {/* the menu's heading, named by its words, says whether the menu shows; the stored isWorkspaceMenuOpen alone
          decides that, in the sidebar and in its copy for peeking alike */}
          <button
            type="button"
            className="flex w-full items-center gap-1 text-left text-13 font-semibold whitespace-nowrap text-placeholder"
            onClick={() => toggleListDisclosure(!isWorkspaceMenuOpen)}
            aria-expanded={!!isWorkspaceMenuOpen}
          >
            <span className="text-13 font-semibold">{t("common.workspace")}</span>
          </button>
          <div className="pointer-events-none flex items-center opacity-0 group-hover:pointer-events-auto group-hover:opacity-100">
            <button
              type="button"
              className="flex-shrink-0 rounded-sm p-0.5 hover:bg-layer-1"
              onClick={() => toggleListDisclosure(!isWorkspaceMenuOpen)}
              // the heading opens and closes the menu as this does: this one, shown to the pointer alone, is neither a
              // Tab stop nor read
              tabIndex={-1}
              aria-hidden="true"
            >
              <ChevronRightOutline
                className={cn("size-3 flex-shrink-0 transition-all", {
                  "rotate-90": isWorkspaceMenuOpen,
                })}
              />
            </button>
          </div>
        </div>
        <Transition
          as="div"
          show={!!isWorkspaceMenuOpen}
          enter="transition duration-100 ease-out"
          enterFrom="transform scale-95 opacity-0"
          enterTo="transform scale-100 opacity-100"
          leave="transition duration-75 ease-out"
          leaveFrom="transform scale-100 opacity-100"
          leaveTo="transform scale-95 opacity-0"
        >
          {isWorkspaceMenuOpen && (
            <div className="flex flex-col gap-0.5">
              {WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS.map((item) => (
                <SidebarItemBase key={item.key} item={item} />
              ))}
            </div>
          )}
        </Transition>
      </div>
    </>
  );
});
