/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import * as React from "react";
import { Menu as BaseMenu } from "@base-ui-components/react/menu";
import { ChevronDownOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
import { cn } from "../utils/classname";
import type { TMenuProps, TMenuItemProps } from "./types";

function MenuItem(props: TMenuItemProps) {
  const { children, disabled = false, onClick, className } = props;

  return (
    <BaseMenu.Item
      disabled={disabled}
      className={cn(
        "w-full cursor-pointer truncate rounded-sm px-1 py-1.5 text-left text-secondary outline-none select-none hover:bg-layer-1",
        {
          "text-placeholder": disabled,
        },
        className
      )}
      onClick={onClick}
    >
      {children}
    </BaseMenu.Item>
  );
}

function Menu(props: TMenuProps) {
  const {
    ariaLabel,
    buttonClassName = "",
    customButtonClassName = "",
    customButtonTabIndex = 0,
    children,
    customButton,
    disabled = false,
    ellipsis = false,
    label,
    maxHeight = "md",
    noBorder = false,
    noChevron = false,
    optionsClassName = "",
    menuItemsClassName = "",
    verticalEllipsis = false,
    menuButtonOnClick,
    onMenuClose,
    tabIndex,
    openOnHover = false,
    handleOpenChange = () => {},
  } = props;

  const [isOpen, setIsOpen] = React.useState(false);

  const openDropdown = () => {
    setIsOpen(true);
  };

  const closeDropdown = React.useCallback(() => {
    if (isOpen) {
      onMenuClose?.();
    }
    setIsOpen(false);
  }, [isOpen, onMenuClose]);

  const handleMenuButtonClick = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.stopPropagation();
    e.preventDefault();
    if (isOpen) {
      closeDropdown();
    } else {
      openDropdown();
    }
    if (menuButtonOnClick) menuButtonOnClick();
  };

  return (
    <BaseMenu.Root openOnHover={openOnHover} onOpenChange={handleOpenChange}>
      {customButton ? (
        <BaseMenu.Trigger
          type="button"
          onClick={handleMenuButtonClick}
          className={cn(customButtonClassName, "outline-none")}
          tabIndex={customButtonTabIndex}
          disabled={disabled}
          aria-label={ariaLabel}
        >
          {customButton}
        </BaseMenu.Trigger>
      ) : (
        <>
          {ellipsis || verticalEllipsis ? (
            <BaseMenu.Trigger
              type="button"
              onClick={handleMenuButtonClick}
              disabled={disabled}
              className={`relative grid place-items-center rounded-sm p-1 text-secondary outline-none hover:text-primary ${
                disabled ? "cursor-not-allowed" : "cursor-pointer hover:bg-layer-1"
              } ${buttonClassName}`}
              tabIndex={customButtonTabIndex}
              aria-label={ariaLabel}
            >
              <MoreHorizontalOutline className={`h-3.5 w-3.5 ${verticalEllipsis ? "rotate-90" : ""}`} />
            </BaseMenu.Trigger>
          ) : (
            <BaseMenu.Trigger
              type="button"
              className={`flex items-center justify-between gap-1 rounded-md px-2.5 py-1 text-11 whitespace-nowrap duration-300 outline-none ${
                isOpen ? "bg-surface-2 text-primary" : "text-secondary"
              } ${noBorder ? "" : "shadow-sm border border-strong focus:outline-none"} ${
                disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer hover:bg-layer-1"
              } ${buttonClassName}`}
              onClick={handleMenuButtonClick}
              tabIndex={customButtonTabIndex}
              disabled={disabled}
              aria-label={ariaLabel}
            >
              {label}
              {!noChevron && <ChevronDownOutline className="h-3.5 w-3.5" />}
            </BaseMenu.Trigger>
          )}
        </>
      )}
      <BaseMenu.Portal>
        <BaseMenu.Positioner
          align={"start"}
          className={cn(
            "fixed z-30 translate-y-0",
            menuItemsClassName
          )} /** translate-y-0 is a hack to create new stacking context. Required for safari  */
        >
          <BaseMenu.Popup
            tabIndex={tabIndex}
            className={cn(
              "my-1 min-w-[12rem] overflow-y-scroll rounded-md border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap shadow-raised-200 focus:outline-none",
              {
                "max-h-60": maxHeight === "lg",
                "max-h-48": maxHeight === "md",
                "max-h-36": maxHeight === "rg",
                "max-h-28": maxHeight === "sm",
              },
              optionsClassName
            )}
            data-main-menu="true"
          >
            {children}
          </BaseMenu.Popup>
        </BaseMenu.Positioner>
      </BaseMenu.Portal>
    </BaseMenu.Root>
  );
}

Menu.MenuItem = MenuItem;

export { Menu };
