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
  /** Says the card of the notification id names holds the focus, element in it taking it: its element's onFocus. */
  focusIn: (id: string, element: EventTarget) => void;
  /** Says element lost the focus: a card's element's onBlur, which ends the card's holding it if element held it. */
  focusOut: (element: EventTarget) => void;
};

/** The card that holds the focus: its notification's id, and the element in it that took the focus last. */
type Holder = { id: string; element: EventTarget };

/**
 * Where the focus goes as a card that holds it leaves the notifications page's list (P4 spec 2.6, R16, R25): by the
 * card's own archive or snooze, preset or custom, which the view no longer lists, or by nerve's notification.not_found.
 * A card holds the focus while it or anything it opened has it, its snooze menu and its dialog too (React's tree,
 * through their portals), until the element that took it last loses it: a blur heard after the focus has come back
 * into the card ends nothing. The dialog gives it back so while its request is out: its submit button, disabled, loses
 * it, and the dialog's listener (keepFocusIn) focuses the dialog in a microtask, which a browser runs before the blur
 * reaches the list (between an event's listeners), jsdom after. Once the card has left, the next card's main button
 * has the focus, else the previous card's, else the list's region: never the page's body, nor what had it before the
 * card's dialog opened, to which the dialog gives it back as it unmounts with its card (Headless UI, a microtask after
 * that unmount's effects: this comes after them).
 * A card that stays (a read change, R8) keeps it; another card leaving moves it not. ids are the cards shown, in order;
 * the list names each card's element by its notification (data-notification-id), its main button its first.
 */
export function useLeavingFocus(ids: readonly string[]): LeavingFocus {
  const region = useRef<HTMLDivElement>(null);
  /** The card that holds the focus. */
  const holder = useRef<Holder | undefined>(undefined);
  /** The cards as the list last showed them. */
  const shown = useRef<readonly string[]>(ids);
  const order = ids.join(" ");
  useEffect(() => {
    const before = shown.current;
    const now = order === "" ? [] : order.split(" ");
    shown.current = now;
    const left = holder.current?.id;
    if (left === undefined || now.includes(left)) return;
    holder.current = undefined;
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
      holder.current = { id, element };
    },
    focusOut: (element) => {
      // a blur heard once the focus has come back into a card ends nothing: only the element that holds it does
      if (holder.current?.element === element) holder.current = undefined;
    },
  };
}
