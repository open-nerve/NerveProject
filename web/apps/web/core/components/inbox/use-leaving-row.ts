/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useRef, type FocusEvent } from "react";

/** What the intake page's list says of the focus: the handlers of the element around it (useLeavingRow). */
type LeavingRow = {
  /** Says what has the focus now, within the list: a row (by its data-intake-row), or anything else. */
  onFocus: (event: FocusEvent) => void;
  /** Says the focus left for its relatedTarget: lost to something, no row holds it; lost to nothing, a row still does. */
  onBlur: (event: FocusEvent) => void;
};

/** Whether nothing has the focus: the document's body has it. */
const nowhere = () => document.activeElement === null || document.activeElement === document.body;

/** The work item of the list's row that target is, or is in (its data-intake-row); none for anything else. */
const rowOf = (target: EventTarget | null): string | undefined =>
  (target instanceof Element ? target.closest("[data-intake-row]")?.getAttribute("data-intake-row") : null) ??
  undefined;

/**
 * Where the focus goes as a row of the intake page's list that has it leaves the list (P5a spec 2.8, R35; the pattern
 * of P4's useLeavingFocus, the notifications' cards, whose card holds and gives back are its own): its work item gone
 * from the tab by a change the page confirmed, such as a triage the user left by Back or Forward while it was out (the
 * selection's detail unmounted with its modal, and Headless UI gave the focus back to the row pressed before). A row
 * holds the focus while it has it, and still once it lost it to nothing (the body: a row removed under it); lost to
 * something, no row does. Once the row has left, nothing having the focus, it goes to the next row, else the previous,
 * else the list (land: the list's region; the detail's region while the list is hidden), a microtask after the commit,
 * never to the page's body. A row that stays keeps it; another row leaving moves it not. ids are the rows shown, in
 * order.
 */
export function useLeavingRow(ids: readonly string[], land: (id: string | undefined) => void): LeavingRow {
  /** The row that holds the focus, by its work item's id. */
  const holder = useRef<string | undefined>(undefined);
  /** The rows as the list last showed them. */
  const shown = useRef<readonly string[]>(ids);
  const order = ids.join(" ");
  useEffect(() => {
    const before = shown.current;
    const now = order === "" ? [] : order.split(" ");
    shown.current = now;
    const left = holder.current;
    if (left === undefined || now.includes(left)) return;
    holder.current = undefined;
    // the focus went elsewhere after the row lost it to nothing: it stays there
    if (!nowhere()) return;
    const at = before.indexOf(left);
    const staying = (id: string) => now.includes(id);
    const next = before.slice(at + 1).find(staying) ?? before.slice(0, Math.max(at, 0)).toReversed().find(staying);
    queueMicrotask(() => land(next));
  }, [order, land]);
  return {
    onFocus: (event) => {
      holder.current = rowOf(event.target);
    },
    onBlur: (event) => {
      if (event.relatedTarget !== null) holder.current = undefined;
    },
  };
}
