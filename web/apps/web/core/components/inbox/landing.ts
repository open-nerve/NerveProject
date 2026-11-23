/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useEffect, useLayoutEffect, useRef, type RefObject } from "react";
import { useSearchParams } from "react-router";

/** Where the intake page's selection leaves a work item for, and where the focus then lands (useIntakeLanding). */
export type IntakeLanding = {
  /** The list's region (the sidebar's), where the focus lands once the selection is cleared, the list shown. */
  list: RefObject<HTMLElement | null>;
  /** The detail's region around the work item selected (InboxContentRoot's), its place while the list is hidden. */
  detail: RefObject<HTMLElement | null>;
  /** The narrow page's button that shows the list, drawn while nothing is selected: its place then, the list hidden. */
  toggle: RefObject<HTMLButtonElement | null>;
  /**
   * Selects the work item to in place of from, or none (undefined), by a replace, while the address still selects
   * from; the focus then lands on to's row in the list, or on the list, while the page shows the list, else on what
   * the page shows instead (useIntakeLanding).
   */
  leave: (from: string, to: string | undefined) => void;
};

/**
 * What the intake page shows: whether its list is on the screen (a wide screen's, or a narrow screen's while shown),
 * and the first row of its list, which the page selects once nothing is.
 */
export type IntakeShown = { list: boolean; first: string | undefined };

/**
 * The selection's leaving a work item, as an action on it ends (P5a spec 2.6, 2.8, R14, R35): a triage nerve confirmed
 * (to the row after the one triaged, else before, as the triage began), a deletion, or a work item the page cannot show
 * (to none). Each is the page's own doing, not the user's: a replace, so that Back never returns to a work item the tab
 * no longer lists; and only while the address still selects the work item left, never over a selection the user
 * changed meanwhile. The focus goes with the selection, never to the page's body nor to anything the screen does not
 * show: once the page shows the selection, while the list is on the screen, to its row (the list's link the address
 * selects, aria-current), else to the list's region; while the list is hidden (a narrow screen, a row pressed), its row
 * and its region off the screen, to the detail's region around the work item selected, else, nothing selected nor to
 * be, to the list's toggle; nothing selected but a first row the page selects next, to that one's detail once shown,
 * never to the toggle that selection removes. It lands a microtask after the commit that shows it, after Headless UI
 * gives the focus back to what had it as a dialog unmounts (a microtask after that unmount's effects).
 */
export function useIntakeLanding(shown: IntakeShown): IntakeLanding {
  const [params, setParams] = useSearchParams();
  const selected = params.get("inboxIssueId") ?? undefined;
  const list = useRef<HTMLElement>(null);
  const detail = useRef<HTMLElement>(null);
  const toggle = useRef<HTMLButtonElement>(null);
  // the address's selection and what the page showed, as last shown, and the selection a leaving waits for
  const addressed = useRef(selected);
  const page = useRef(shown);
  const landing = useRef<{ to: string | undefined }>(undefined);
  useLayoutEffect(() => {
    addressed.current = selected;
    page.current = shown;
  });
  useEffect(() => {
    const awaited = landing.current;
    if (!awaited || awaited.to !== selected) return;
    const { list: listed, first } = page.current;
    // nothing selected, the list hidden: the page selects its first row next, whose detail the focus lands on then
    if (selected === undefined && !listed && first !== undefined) {
      landing.current = { to: first };
      return;
    }
    landing.current = undefined;
    queueMicrotask(() => {
      const place = listed
        ? ((selected === undefined ? null : list.current?.querySelector<HTMLElement>('[aria-current="true"]')) ??
          list.current)
        : (detail.current ?? toggle.current);
      place?.focus();
    });
  }, [selected]);
  const leave = useCallback(
    (from: string, to: string | undefined) => {
      if (addressed.current !== from) return;
      landing.current = { to };
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (to === undefined) next.delete("inboxIssueId");
          else next.set("inboxIssueId", to);
          return next;
        },
        { replace: true }
      );
    },
    [setParams]
  );
  return { list, detail, toggle, leave };
}
