/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useRef } from "react";
import type { RefObject } from "react";

/** The list's region, and what its cards say of the focus (useLeavingFocus). */
type LeavingFocus = {
  /** The element of the list there whatever it shows, cards or none: the focus's place once no card is left. */
  region: RefObject<HTMLDivElement | null>;
  /**
   * Says the card of the notification id names holds the focus, which element has: its element's onFocus. The control
   * it is, or the button of the menu it is in (controlOf), has the focus back.
   */
  focusIn: (id: string, element: EventTarget) => void;
  /**
   * Says an element of a card lost the focus to to (null: to nothing): a card's element's onBlur. Lost to something, no
   * card holds it; lost to nothing, the card holds it still.
   */
  focusOut: (to: EventTarget | null) => void;
};

/** Whether nothing has the focus: the document's body has it. */
const nowhere = () => document.activeElement === null || document.activeElement === document.body;

/**
 * The control that element, in a card and having the focus, gives it back to: element itself; for anything in a menu
 * (its panel, or one of its items, which the pointer's press focuses), which the menu takes away as it closes, the
 * menu's button, which the menu names (aria-labelledby).
 */
function controlOf(element: HTMLElement): HTMLElement | null {
  const menu = element.closest('[role="menu"]');
  return menu === null ? element : document.getElementById(menu.getAttribute("aria-labelledby") ?? "");
}

/**
 * Where the focus goes as a card that holds it leaves the notifications page's list (P4 spec 2.6, R16, R25): by the
 * card's own archive or snooze, preset or custom, which the view no longer lists, or by nerve's notification.not_found.
 * A card holds the focus while it or anything it opened has it, its snooze menu and its dialog too (React's tree,
 * through their portals), until it goes elsewhere. Losing it to nothing, the document's body having it, does not end
 * its holding it: a control disabled while its change is out, which a browser blurs (Chromium and Firefox, WebKit when
 * it was pressed from the keyboard), the card's actions and its dialog's submit button; the snooze menu, which goes
 * with the focus as a preset is chosen, its button disabled by then, so that Headless UI cannot give it back. The
 * submit button's blur reaches the list after the dialog has the focus again (keepFocusIn, a microtask that a browser
 * runs between an event's listeners, jsdom after). The card has it back while nothing else has it: once the card has
 * left, the next card's main button has the focus, else the previous card's, else the list's region; never the page's
 * body, nor what had it before the card's dialog opened, to which the dialog gives it back as it unmounts with its card
 * (Headless UI, a microtask after that unmount's effects: this comes after them). The card having stayed (a read
 * change, R8; a refusal), the control that had the focus last, or the button of the menu it was in, has it back once
 * it is enabled again while nothing has it (O4): a preset chosen by the keyboard or by the pointer (whose press
 * focuses the item, never the menu's button) gives it back to the snooze menu's button. A card that stays keeps it;
 * another card leaving moves it not. ids are the cards shown, in order; the list names each card's element by its
 * notification (data-notification-id), its main button its first.
 */
export function useLeavingFocus(ids: readonly string[]): LeavingFocus {
  const region = useRef<HTMLDivElement>(null);
  /** The card that holds the focus, by its notification's id. */
  const holder = useRef<string | undefined>(undefined);
  /**
   * The control that had the focus last (controlOf), and the watch by which it has it back once it is enabled again
   * while nothing has it.
   */
  const returning = useRef<{ control: HTMLElement; watch: MutationObserver } | undefined>(undefined);
  const stopReturning = () => {
    returning.current?.watch.disconnect();
    returning.current = undefined;
  };
  useEffect(() => stopReturning, []);
  /** The cards as the list last showed them. */
  const shown = useRef<readonly string[]>(ids);
  const order = ids.join(" ");
  useEffect(() => {
    const before = shown.current;
    const now = order === "" ? [] : order.split(" ");
    shown.current = now;
    // the control watched has gone with its card, whichever card holds the focus now: it is watched no more
    if (returning.current?.control.isConnected === false) stopReturning();
    const left = holder.current;
    if (left === undefined || now.includes(left)) return;
    holder.current = undefined;
    stopReturning();
    // the focus went elsewhere after the card lost it to nothing: it stays there
    if (!nowhere()) return;
    const at = before.indexOf(left);
    const staying = (id: string) => now.includes(id);
    const next = before.slice(at + 1).find(staying) ?? before.slice(0, Math.max(at, 0)).toReversed().find(staying);
    queueMicrotask(() => {
      const card =
        next === undefined
          ? null
          : region.current?.querySelector<HTMLElement>(`[data-notification-id="${next}"] button`);
      (card ?? region.current)?.focus();
    });
  }, [order]);
  return {
    region,
    focusIn: (id, element) => {
      holder.current = id;
      const control = element instanceof HTMLElement ? controlOf(element) : null;
      if (control === null) return;
      stopReturning();
      const watch = new MutationObserver(() => {
        if (!control.matches(":disabled") && control.isConnected && nowhere()) control.focus();
      });
      watch.observe(control, { attributes: true, attributeFilter: ["disabled"] });
      returning.current = { control, watch };
    },
    focusOut: (to) => {
      // lost to something: no card holds it; lost to nothing, the card holds it still
      if (to !== null) holder.current = undefined;
    },
  };
}
