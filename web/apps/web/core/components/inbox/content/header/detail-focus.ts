/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useRef, useState, type RefObject } from "react";

/** Where the focus lands in the detail: the control that took the action, or the status's primary action. */
type Landing = "opener" | "primary";

/**
 * The control an element that took an action gives the focus back to (P4's lesson O4): the element itself; for an item
 * of a menu, which the menu takes away as it closes (and which the pointer's press focuses, never the menu's button),
 * the menu's button, which the menu names (aria-labelledby, Headless UI's).
 */
function controlOf(element: HTMLElement): HTMLElement | null {
  const menu = element.closest('[role="menu"]');
  return menu === null ? element : document.getElementById(menu.getAttribute("aria-labelledby") ?? "");
}

/**
 * The primary actions the detail's headers draw (data-intake-primary): the wide header's button, the narrow header's
 * menu (its button), the one a screen this narrow or wide shows taking the focus.
 */
const primaryActions = (region: HTMLElement | null): (HTMLElement | null)[] =>
  [...(region?.querySelectorAll<HTMLElement>("[data-intake-primary]") ?? [])].map((marked) =>
    marked instanceof HTMLButtonElement ? marked : marked.querySelector("button")
  );

/** Gives the focus to the first of elements that takes it: none gone, hidden (the other header's) or disabled. */
function focusFirst(elements: (HTMLElement | null)[]): void {
  for (const element of elements) {
    if (!element?.isConnected) continue;
    element.focus();
    if (document.activeElement === element) return;
  }
}

/**
 * Where the focus lands in the selected work item's detail as an action on it ends with the selection staying (P5a
 * spec 2.8, R35; re-94 I4), never on the page's body: back on the control that took it (a cancel; a refusal of an
 * action no modal holds), once it is enabled again; else, that control gone with the header a triage's read again
 * withdrew (a duplicate marked, a conflict), on the new header's primary action, else on the detail's region. It lands
 * a microtask after the commit that ends the action, the header redrawn and its controls enabled, and after Headless
 * UI gives the focus back to what had it as a dialog unmounts (a microtask after that unmount's effects).
 */
export function useDetailFocus(region: RefObject<HTMLElement | null>) {
  const opener = useRef<HTMLElement | null>(null);
  const [landing, setLanding] = useState<{ on: Landing }>();
  useEffect(() => {
    if (!landing) return;
    queueMicrotask(() =>
      focusFirst([landing.on === "opener" ? opener.current : null, ...primaryActions(region.current), region.current])
    );
  }, [landing, region]);
  return {
    /** Remembers the control that took an action: element, or its menu's button. */
    remember: (element: HTMLElement | null | undefined) => {
      opener.current = element ? controlOf(element) : null;
    },
    /** Lands the focus, once the action's end is shown: on the control that took it, or on the primary action. */
    land: (on: Landing) => setLanding({ on }),
  };
}
