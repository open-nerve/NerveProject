/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useEffect, useLayoutEffect, useRef, type RefObject } from "react";
// local imports
import { useNotificationPageState } from "./page-state";

/** Where the notifications page's preview leaves its notification, and its region (useNotificationLanding). */
export type NotificationLanding = {
  /** The preview's region (the section named Preview), the focus's last place. */
  preview: RefObject<HTMLElement | null>;
  /**
   * Clears the selection of the notification notificationId, while it is the one selected; the focus then lands on
   * its card's main button, else on the list's region, else on the preview's region.
   */
  leave: (notificationId: string) => void;
};

/**
 * The notifications page's landing of the focus as its preview leaves the notification selected (P5b spec 2.5, R26;
 * re-95 I3, O7): the embed's close, its deletion and a work item it cannot show; the close of the preview's failure;
 * the plain peek's close. Only while the selection is still that notification, the selection is cleared; a microtask
 * after the commit that clears it (after Headless UI gives the focus back to what had it as a dialog unmounts, a
 * microtask after that unmount's effects), the focus lands on the main button of the notification's card (its first,
 * the list's own way: data-notification-id), else, the card gone (the selection stays as the tab and the filters
 * change), on the list's region (data-notification-list), else, the list not drawn (its read failed or is out), on the
 * preview's region; never on the page's body. NotificationsRoot holds it, which the preview's leaving never unmounts.
 */
export function useNotificationLanding(): NotificationLanding {
  const { selected, clearSelection } = useNotificationPageState();
  const preview = useRef<HTMLElement>(null);
  // the notification selected as last shown, and the one a leaving cleared, whose card the focus lands on
  const shown = useRef(selected?.notificationId);
  const leaving = useRef<string | undefined>(undefined);
  useLayoutEffect(() => {
    shown.current = selected?.notificationId;
  });
  useEffect(() => {
    const left = leaving.current;
    if (left === undefined) return;
    leaving.current = undefined;
    // another notification selected meanwhile: the focus is that one's
    if (selected !== undefined) return;
    queueMicrotask(() => {
      const list = document.querySelector<HTMLElement>("[data-notification-list]");
      const card = list?.querySelector<HTMLElement>(`[data-notification-id="${left}"] button`);
      (card ?? list ?? preview.current)?.focus();
    });
  }, [selected]);
  const leave = useCallback(
    (notificationId: string) => {
      if (shown.current !== notificationId) return;
      leaving.current = notificationId;
      clearSelection(notificationId);
    },
    [clearSelection]
  );
  return { preview, leave };
}
