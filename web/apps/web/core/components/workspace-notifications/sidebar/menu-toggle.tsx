/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { TickOutline } from "@makeplane/propel/icons";
// nerve imports
import { cn } from "@nerve/utils";

type Props = {
  /** What the item turns on or off, in the page's language. */
  label: string;
  /** Whether it is on. */
  pressed: boolean;
  /** Turns it off if on, on if off. */
  onToggle: () => void;
};

/**
 * An item of the notifications page's head menus (P4 spec 2.6, R16, re-93 I3): a plain button that says whether it is
 * pressed. A menu of them is a PopoverMenu, which has no arrow keys: Tab reaches each item, and Enter and Space toggle it
 * once each, as a button's own keys; it has no key handler of its own, which would toggle it a second time. As a button
 * of its own type it never submits a form around it.
 */
export function NotificationMenuToggle({ label, pressed, onToggle }: Props) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      className="flex w-full cursor-pointer items-center gap-2 rounded-xs p-1 px-2 text-13 transition-all hover:bg-layer-1 focus-visible:outline-offset-2"
      onClick={onToggle}
    >
      <span
        aria-hidden="true"
        className={cn(
          "flex h-3 w-3 flex-shrink-0 items-center justify-center rounded-xs transition-all",
          pressed ? "bg-accent-primary text-on-color" : "bg-surface-2"
        )}
      >
        {pressed && <TickOutline className="h-2.5 w-2.5" />}
      </span>
      <span className={cn("whitespace-nowrap", pressed ? "text-primary" : "text-secondary")}>{label}</span>
    </button>
  );
}
