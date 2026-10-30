/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import ReactDOM from "react-dom";
import { isComposing } from "@nerve/utils";
import { usePlatformOS } from "../../hooks/use-platform-os";
import { cn } from "../../utils";
import { ContextMenuItem } from "./item";

export type TContextMenuItem = {
  key: string;
  customContent?: React.ReactNode;
  title?: string;
  description?: string;
  icon?: React.ComponentType<{ className?: string }>;
  action: () => void;
  shouldRender?: boolean;
  closeOnClick?: boolean;
  disabled?: boolean;
  className?: string;
  iconClassName?: string;
};
type ContextMenuProps = {
  parentRef: React.RefObject<HTMLElement | null>;
  items: TContextMenuItem[];
  portalContainer?: Element | null;
};
function enabledItems(panel: HTMLElement) {
  return Array.from(panel.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)'));
}
function ContextMenuWithoutPortal({ parentRef, items }: ContextMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const contextMenuRef = useRef<HTMLDivElement>(null);
  const returnFocus = useRef<HTMLElement | null>(null);
  const { isMobile } = usePlatformOS();
  const handleClose = useCallback(() => {
    setIsOpen(false);
    if (returnFocus.current?.isConnected) returnFocus.current.focus();
  }, []);
  useEffect(() => {
    const parent = parentRef.current,
      panel = contextMenuRef.current;
    if (!parent || !panel) return;
    const open = (x: number, y: number) => {
      returnFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : parent;
      setPosition({
        x: Math.max(0, Math.min(x, window.innerWidth - panel.clientWidth)),
        y: Math.max(0, Math.min(y, window.innerHeight - panel.clientHeight)),
      });
      setIsOpen(true);
    };
    const contextmenu = (event: MouseEvent) => {
      if (isMobile) return;
      event.preventDefault();
      event.stopPropagation();
      open(event.clientX, event.clientY);
    };
    const keyboard = (event: KeyboardEvent) => {
      if (isComposing(event) || !(event.key === "ContextMenu" || (event.shiftKey && event.key === "F10"))) return;
      event.preventDefault();
      event.stopPropagation();
      const box = parent.getBoundingClientRect();
      open(box.left, box.bottom);
    };
    parent.addEventListener("contextmenu", contextmenu);
    parent.addEventListener("keydown", keyboard);
    return () => {
      parent.removeEventListener("contextmenu", contextmenu);
      parent.removeEventListener("keydown", keyboard);
    };
  }, [isMobile, parentRef]);
  useLayoutEffect(() => {
    if (isOpen && contextMenuRef.current) (enabledItems(contextMenuRef.current)[0] ?? contextMenuRef.current).focus();
  }, [isOpen]);
  useEffect(() => {
    if (!isOpen) return;
    const outside = (event: MouseEvent) => {
      if (event.target instanceof Node && !contextMenuRef.current?.contains(event.target)) setIsOpen(false);
    };
    document.addEventListener("mousedown", outside, true);
    return () => document.removeEventListener("mousedown", outside, true);
  }, [isOpen]);
  return (
    // Kept for measurement. Closed, no item participates in Tab or the accessibility tree.
    <div
      inert={!isOpen}
      aria-hidden={!isOpen}
      className={cn(
        "pointer-events-none fixed top-0 left-0 z-30 h-screen w-screen cursor-default opacity-0 transition-opacity",
        { "pointer-events-auto opacity-100": isOpen }
      )}
    >
      <div
        ref={contextMenuRef}
        role="menu"
        tabIndex={-1}
        className="vertical-scrollbar fixed scrollbar-sm max-h-72 min-w-[12rem] overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5"
        style={{ top: position.y, left: position.x }}
        data-context-menu="true"
        onKeyDown={(event) => {
          if (isComposing(event.nativeEvent)) {
            event.preventDefault();
            event.stopPropagation();
            return;
          }
          if (event.key === "Escape") {
            event.preventDefault();
            event.stopPropagation();
            handleClose();
            return;
          }
          if (event.key === "Tab") {
            setIsOpen(false);
            return;
          }
          const buttons = enabledItems(event.currentTarget);
          const current = buttons.findIndex((button) => button === document.activeElement);
          const next =
            event.key === "ArrowDown"
              ? (current + 1) % buttons.length
              : event.key === "ArrowUp"
                ? (current - 1 + buttons.length) % buttons.length
                : event.key === "Home"
                  ? 0
                  : event.key === "End"
                    ? buttons.length - 1
                    : undefined;
          if (next !== undefined) {
            event.preventDefault();
            event.stopPropagation();
            buttons[next]?.focus();
          }
        }}
      >
        {items
          .filter((item) => item.shouldRender !== false)
          .map((item) => (
            <ContextMenuItem key={item.key} item={item} handleClose={handleClose} />
          ))}
      </div>
    </div>
  );
}
export function ContextMenu(props: ContextMenuProps) {
  const menu = <ContextMenuWithoutPortal {...props} />;
  const portal = props.portalContainer ?? document.querySelector("#context-menu-portal");
  return portal ? ReactDOM.createPortal(menu, portal) : menu;
}
