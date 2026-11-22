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
  /** Says the card of the notification id names holds the focus: its element's onFocus. */
  focusIn: (id: string) => void;
  /**
   * Says element, in a card, lost the focus to to (null: to nothing): a card's element's onBlur. Lost to something, no
   * card holds it; lost to nothing, the card holds it still.
   */
  focusOut: (element: EventTarget, to: EventTarget | null) => void;
};

/** Whether nothing has the focus: the document's body has it. */
const nowhere = () => document.activeElement === null || document.activeElement === document.body;

/**
 * Where the focus goes as a card that holds it leaves the notifications page's list (P4 spec 2.6, R16, R25): by the
 * card's own archive or snooze, preset or custom, which the view no longer lists, or by nerve's notification.not_found.
 * A card holds the focus while it or anything it opened has it, its snooze menu and its dialog too (React's tree,
 * through their portals), until it goes elsewhere. A control of the card that loses it to nothing, the document's body
 * having it, does not end its holding it: a control disabled while its change is out, which a browser blurs (Chromium
 * and Firefox, WebKit when it was pressed from the keyboard), the card's actions and its dialog's submit button. That
 * button's blur reaches the list after the dialog has the focus again (keepFocusIn, a microtask that a browser runs
 * between an event's listeners, jsdom after). The card has it back while nothing else has it: once the card has left,
 * the next card's main button has the focus, else the previous card's, else the list's region; never the page's body,
 * nor what had it before the card's dialog opened, to which the dialog gives it back as it unmounts with its card
 * (Headless UI, a microtask after that unmount's effects: this comes after them). Once the control is enabled again,
 * the card having stayed (a read change, R8), it has the focus back. A card that stays keeps it; another card leaving
 * moves it not. ids are the cards shown, in order; the list names each card's element by its notification
 * (data-notification-id), its main button its first.
 */
export function useLeavingFocus(ids: readonly string[]): LeavingFocus {
  const region = useRef<HTMLDivElement>(null);
  /** The card that holds the focus, by its notification's id. */
  const holder = useRef<string | undefined>(undefined);
  /** The watch of the control that lost the focus to nothing, which has it back once it is enabled again. */
  const returning = useRef<MutationObserver | undefined>(undefined);
  const stopReturning = () => {
    returning.current?.disconnect();
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
    const left = holder.current;
    if (left === undefined || now.includes(left)) return;
    holder.current = undefined;
    stopReturning();
    // the focus went elsewhere after the card's control lost it to nothing: it stays there
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
    focusIn: (id) => {
      stopReturning();
      holder.current = id;
    },
    focusOut: (element, to) => {
      stopReturning();
      if (to !== null || !(element instanceof HTMLElement)) {
        holder.current = undefined;
        return;
      }
      // lost to nothing: the card holds it still, and the control has it back once enabled again
      const watch = new MutationObserver(() => {
        if (element.matches(":disabled")) return;
        stopReturning();
        if (element.isConnected && nowhere()) element.focus();
      });
      watch.observe(element, { attributes: true, attributeFilter: ["disabled"] });
      returning.current = watch;
    },
  };
}
