/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { problemCode } from "@/lib/api-error";
// local imports
import { useNotificationPageState } from "../page-state";

/**
 * How a card says nerve's refusal of a change of its notification (P4 spec 2.6, re-93 I4), opening it read or one of
 * its actions: a toast says why (useRefusalToast); for notification.not_found, after which the notification has left
 * every list (the store's update), the page's selection of it is cleared too, so that the preview never stays on a
 * notification nerve no longer gives (another selected meanwhile stays). A caller follows the change in the session it
 * was sent in (followInSession), and calls this with nerve's refusal.
 */
export function useNotificationRefusal(notificationId: string): (error: unknown) => void {
  const { clearSelection } = useNotificationPageState();
  const refusalToast = useRefusalToast();
  return (error) => {
    refusalToast(error);
    if (problemCode(error) === "notification.not_found") clearSelection(notificationId);
  };
}
