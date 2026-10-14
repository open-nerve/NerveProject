/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Menu } from "@headlessui/react";
import { ChevronDownOutline, ChevronRightOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
import * as React from "react";
import ReactDOM from "react-dom";
import { usePopper } from "react-popper";
// helpers
import { cn } from "../utils";
// types
import type { ICustomMenuDropdownProps, ICustomMenuItemProps, ICustomSubMenuProps } from "./helper";

interface PortalProps {
  children: React.ReactNode;
  container?: Element | null;
  asChild?: boolean;
}

function Portal({ children, container, asChild = false }: PortalProps) {
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
    return () => setMounted(false);
  }, []);

  if (!mounted) {
    return null;
  }

  const targetContainer = container || document.body;

  if (asChild) {
    return ReactDOM.createPortal(children, targetContainer);
  }

  return ReactDOM.createPortal(<div data-radix-portal="">{children}</div>, targetContainer);
}

// Context for main menu to communicate with submenus: closing the others as one opens, and the menu's items element,
// which holds what they show
const MenuContext = React.createContext<{
  closeAllSubmenus: () => void;
  registerSubmenu: (closeSubmenu: () => void) => () => void;
  itemsElement: HTMLElement | null;
} | null>(null);

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
  const [popperElement, setPopperElement] = React.useState<HTMLDivElement | null>(null);
  const [itemsElement, setItemsElement] = React.useState<HTMLElement | null>(null);
  // refs
  const submenuClosersRef = React.useRef<Set<() => void>>(new Set());

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "auto",
  });

  const closeAllSubmenus = React.useCallback(() => {
    submenuClosersRef.current.forEach((closeSubmenu) => closeSubmenu());
  }, []);

  const registerSubmenu = React.useCallback((closeSubmenu: () => void) => {
    submenuClosersRef.current.add(closeSubmenu);
    return () => {
      submenuClosersRef.current.delete(closeSubmenu);
    };
  }, []);

  const menuContextValue = React.useMemo(
    () => ({ closeAllSubmenus, registerSubmenu, itemsElement }),
    [closeAllSubmenus, registerSubmenu, itemsElement]
  );

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
      data-main-menu="true"
    >
      {({ open, close }) => {
        let menuItems = (
          <Menu.Items
            ref={setItemsElement}
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
              ref={setPopperElement}
              style={styles.popper}
              {...attributes.popper}
            >
              <MenuContext.Provider value={menuContextValue}>{children}</MenuContext.Provider>
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

// SubMenu context for closing submenu from nested items
const SubMenuContext = React.createContext<{ closeSubmenu: () => void } | null>(null);

// Hook to use submenu context
const useSubMenu = () => React.useContext(SubMenuContext);

// SubMenu implementation
function SubMenu(props: ICustomSubMenuProps) {
  const {
    children,
    trigger,
    disabled = false,
    className = "",
    contentClassName = "",
    placement = "right-start",
  } = props;

  const [isOpen, setIsOpen] = React.useState(false);
  const [referenceElement, setReferenceElement] = React.useState<HTMLSpanElement | null>(null);
  const [popperElement, setPopperElement] = React.useState<HTMLDivElement | null>(null);
  const submenuRef = React.useRef<HTMLDivElement | null>(null);

  const menuContext = React.useContext(MenuContext);

  // in the menu's items element, which is fixed, no box around it clips what this shows: the viewport alone bounds it
  // (no clipping parents)
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement,
    modifiers: [
      {
        name: "offset",
        options: {
          offset: [0, 4],
        },
      },
      {
        name: "flip",
        options: {
          fallbackPlacements: ["left-start", "right-end", "left-end", "top-start", "bottom-start"],
          boundary: [],
        },
      },
      {
        name: "preventOverflow",
        options: {
          padding: 8,
          boundary: [],
        },
      },
    ],
  });

  const closeSubmenu = React.useCallback(() => {
    setIsOpen(false);
  }, []);

  // Register this submenu with the main menu context
  React.useEffect(() => {
    if (menuContext) {
      return menuContext.registerSubmenu(closeSubmenu);
    }
  }, [menuContext, closeSubmenu]);

  const toggleSubmenu = () => {
    if (!disabled) {
      // Close other submenus when opening this one
      if (!isOpen && menuContext) {
        menuContext.closeAllSubmenus();
      }
      setIsOpen(!isOpen);
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    toggleSubmenu();
  };

  const subMenuContextValue = React.useMemo(() => ({ closeSubmenu }), [closeSubmenu]);

  // Close submenu when clicking on other menu items
  React.useEffect(() => {
    const handleMenuItemClick = (e: Event) => {
      const target = e.target as HTMLElement;
      // Check if the click is on a menu item that's not part of this submenu
      if (target.closest('[role="menuitem"]') && !submenuRef.current?.contains(target)) {
        closeSubmenu();
      }
    };

    document.addEventListener("click", handleMenuItemClick);
    return () => {
      document.removeEventListener("click", handleMenuItemClick);
    };
  }, [closeSubmenu]);

  return (
    <div ref={submenuRef} className={cn("relative", className)}>
      <span ref={setReferenceElement} className="w-full">
        <Menu.Item as="div" disabled={disabled}>
          {({ active }) => (
            <button
              type="button"
              className={cn(
                "font-inherit flex w-full cursor-pointer items-center justify-between rounded-sm border-0 bg-transparent px-1 py-1.5 text-left text-secondary outline-none select-none",
                {
                  "bg-layer-transparent-hover": active && !disabled,
                  "text-placeholder": disabled,
                  "cursor-not-allowed": disabled,
                }
              )}
              onClick={handleClick}
              disabled={disabled}
            >
              <span className="flex-1">{trigger}</span>
              <ChevronRightOutline className="h-3.5 w-3.5 flex-shrink-0" />
            </button>
          )}
        </Menu.Item>
      </span>

      {isOpen && (
        // in the menu's items element, beside its list, whose scrolling does not clip it: a click on it is a click in
        // the menu, which Headless UI's menu stays open for
        <Portal container={menuContext?.itemsElement}>
          <div
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className={cn(
              "shadow-md z-30 min-w-[12rem] overflow-hidden rounded-md border border-strong-1 bg-surface-1 p-1 text-11 ring-1 ring-strong-1/15",
              contentClassName
            )}
            data-prevent-outside-click="true"
          >
            <SubMenuContext.Provider value={subMenuContextValue}>{children}</SubMenuContext.Provider>
          </div>
        </Portal>
      )}
    </div>
  );
}

function MenuItem(props: ICustomMenuItemProps) {
  const { children, disabled = false, onClick, className } = props;
  const submenuContext = useSubMenu();

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
            // Close submenu if this item is inside a submenu
            submenuContext?.closeSubmenu();
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
CustomMenu.SubMenu = SubMenu;

export { CustomMenu };
