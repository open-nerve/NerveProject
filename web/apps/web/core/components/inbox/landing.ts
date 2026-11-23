/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useEffect, useLayoutEffect, useRef, type RefObject } from "react";
import { useSearchParams } from "react-router";

/** Where the intake page's selection leaves a work item for, and where the focus then lands (useIntakeLanding). */
export type IntakeLanding = {
  /** The list's region (the sidebar's), where the focus lands once the selection is cleared. */
  list: RefObject<HTMLElement | null>;
  /**
   * Selects the work item to in place of from, or none (undefined), by a replace, while the address still selects
   * from; the focus then lands on to's row in the list, or on the list.
   */
  leave: (from: string, to: string | undefined) => void;
};

/**
 * The selection's leaving a work item, as an action on it ends (P5a spec 2.6, 2.8, R14, R35): a triage nerve confirmed
 * (to the row after the one triaged, else before, as the triage began), a deletion, or a work item the page cannot show
 * (to none). Each is the page's own doing, not the user's: a replace, so that Back never returns to a work item the tab
 * no longer lists; and only while the address still selects the work item left, never over a selection the user
 * changed meanwhile. The focus goes with the selection, never to the page's body: once the page shows the selection, to
 * its row (the list's link the address selects, aria-current), else to the list's region; a microtask after the commit
 * that shows it, after Headless UI gives the focus back to what had it as a dialog unmounts (a microtask after that
 * unmount's effects).
 */
export function useIntakeLanding(): IntakeLanding {
  const [params, setParams] = useSearchParams();
  const selected = params.get("inboxIssueId") ?? undefined;
  const list = useRef<HTMLElement>(null);
  // the address's selection as last shown, and the selection a leaving waits for, to land the focus then
  const shown = useRef(selected);
  const landing = useRef<{ to: string | undefined }>(undefined);
  useLayoutEffect(() => {
    shown.current = selected;
  }, [selected]);
  useEffect(() => {
    const awaited = landing.current;
    if (!awaited || awaited.to !== selected) return;
    landing.current = undefined;
    const region = list.current;
    queueMicrotask(() => {
      const row = selected === undefined ? null : region?.querySelector<HTMLElement>('[aria-current="true"]');
      (row ?? region)?.focus();
    });
  }, [selected]);
  const leave = useCallback(
    (from: string, to: string | undefined) => {
      if (shown.current !== from) return;
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
  return { list, leave };
}
