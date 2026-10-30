/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Menu } from "@headlessui/react";
import { ChevronDownOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
import * as React from "react";
import { isComposing } from "@nerve/utils";
import ReactDOM from "react-dom";
import { usePopper } from "react-popper";
// helpers
import { useInsideModal } from "../modals/modal-core";
import { cn } from "../utils";
// types
import type { ICustomMenuDropdownProps, ICustomMenuItemProps } from "./helper";

/** Calls onOpen as the menu opens and onClose as it closes, once for each change of open. */
function OpenChange(props: { open: boolean; onOpen?: () => void; onClose?: () => void }) {
  const { open, onOpen, onClose } = props;
  const wasOpen = React.useRef(open);

  React.useEffect(() => {
    if (open === wasOpen.current) return;
    wasOpen.current = open;
    if (open) onOpen?.();
    else onClose?.();
  }, [open, onOpen, onClose]);

  return null;
}

/**
 * A menu: Headless UI's Menu, whose open state is the one it shows. Its button opens it by the pointer, Enter, Space or
 * the arrow keys; the arrow keys move among its items, Enter and Space choose one. A choice or Escape closes it, the
 * focus going back to its button; Tab closes it and moves on from its button; a click outside closes it too.
 */
function CustomMenu(props: ICustomMenuDropdownProps) {
  const {
    ariaLabel,
    buttonClassName = "",
    customButtonClassName = "",
    customButtonTabIndex = 0,
    placement,
    children,
    className = "",
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
    portalElement,
    menuButtonOnClick,
    onMenuClose,
    tabIndex,
    closeOnSelect,
  } = props;

  const [referenceElement, setReferenceElement] = React.useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = React.useState<HTMLElement | null>(null);
  const insideModal = useInsideModal();
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "auto",
    strategy: "fixed",
  });

  // the menu's place in the Tab order, where its caller gives one, is its button's: the button is the menu's one stop
  const buttonTabIndex = tabIndex ?? customButtonTabIndex;

  return (
    <Menu
      as="div"
      className={cn("relative w-min text-left", className)}
      role="presentation"
      // a click on the menu's button or items is the menu's alone, not the row's or the link's around it
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
      }}
      onKeyDownCapture={(event) => {
        if (isComposing(event.nativeEvent)) {
          event.preventDefault();
          event.stopPropagation();
        }
      }}
      data-main-menu="true"
    >
      {({ open, close }) => {
        let menuItems = (
          <Menu.Items
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            onKeyDownCapture={(event) => {
              if (isComposing(event.nativeEvent)) {
                event.preventDefault();
                event.stopPropagation();
              }
            }}
            modal={!insideModal}
            data-prevent-outside-click={!!portalElement}
            className={cn(
              "fixed z-30 translate-y-0",
              menuItemsClassName
            )} /** translate-y-0 is a hack to create new stacking context. Required for safari  */
            // an item closes the menu as it is chosen; with closeOnSelect, a click on anything else in it does too
            onClick={closeOnSelect ? () => close() : undefined}
          >
            <div
              className={cn(
                "shadow-md my-1 min-w-[12rem] overflow-y-scroll rounded-md border border-strong-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap ring-1 ring-strong-1/15 outline-none focus:outline-none",
                {
                  "max-h-60": maxHeight === "lg",
                  "max-h-48": maxHeight === "md",
                  "max-h-36": maxHeight === "rg",
                  "max-h-28": maxHeight === "sm",
                },
                optionsClassName
              )}
            >
              {children}
            </div>
          </Menu.Items>
        );

        if (portalElement) {
          menuItems = ReactDOM.createPortal(menuItems, portalElement);
        }

        return (
          <>
            <OpenChange open={open} onOpen={menuButtonOnClick} onClose={onMenuClose} />
            {customButton ? (
              <Menu.Button as={React.Fragment} disabled={disabled}>
                <button
                  ref={setReferenceElement}
                  type="button"
                  className={customButtonClassName}
                  tabIndex={buttonTabIndex}
                  disabled={disabled}
                  aria-label={ariaLabel}
                >
                  {customButton}
                </button>
              </Menu.Button>
            ) : (
              <>
                {ellipsis || verticalEllipsis ? (
                  <Menu.Button as={React.Fragment} disabled={disabled}>
                    <button
                      ref={setReferenceElement}
                      type="button"
                      disabled={disabled}
                      className={`relative grid place-items-center rounded-sm p-1 text-secondary hover:text-primary ${
                        disabled ? "cursor-not-allowed" : "cursor-pointer hover:bg-layer-transparent-hover"
                      } ${buttonClassName}`}
                      tabIndex={buttonTabIndex}
                      aria-label={ariaLabel}
                    >
                      <MoreHorizontalOutline className={`h-3.5 w-3.5 ${verticalEllipsis ? "rotate-90" : ""}`} />
                    </button>
                  </Menu.Button>
                ) : (
                  <Menu.Button as={React.Fragment} disabled={disabled}>
                    <button
                      ref={setReferenceElement}
                      type="button"
                      className={`flex items-center justify-between gap-1 rounded-md px-2.5 py-1 text-11 whitespace-nowrap duration-300 ${
                        open ? "text-primary" : "text-secondary"
                      } ${noBorder ? "" : "shadow-sm border border-strong focus:outline-none"} ${
                        disabled
                          ? "cursor-not-allowed text-secondary"
                          : "cursor-pointer hover:bg-layer-transparent-hover"
                      } ${buttonClassName}`}
                      tabIndex={buttonTabIndex}
                      disabled={disabled}
                      aria-label={ariaLabel}
                    >
                      {label}
                      {!noChevron && <ChevronDownOutline className="h-3.5 w-3.5" />}
                    </button>
                  </Menu.Button>
                )}
              </>
            )}
            {menuItems}
          </>
        );
      }}
    </Menu>
  );
}

function MenuItem(props: ICustomMenuItemProps) {
  const { children, disabled = false, onClick, className } = props;

  return (
    // the item is the button that acts: Enter and Space on the menu click its active item, which chooses it
    <Menu.Item disabled={disabled}>
      {({ focus, close }) => (
        <button
          type="button"
          className={cn(
            "w-full truncate rounded-sm px-1 py-1.5 text-left text-secondary select-none",
            {
              "bg-layer-transparent-hover": focus && !disabled,
              "text-placeholder": disabled,
            },
            className
          )}
          onClick={(e) => {
            close();
            onClick?.(e);
          }}
          disabled={disabled}
        >
          {children}
        </button>
      )}
    </Menu.Item>
  );
}

// Add all components as static properties for external use
CustomMenu.MenuItem = MenuItem;

export { CustomMenu };
