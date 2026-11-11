/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef } from "react";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";

/** What a list page whose preferences are its own tells the follow of a change of them (P5b spec 2.4). */
type PreferencesPage = {
  /** Takes the page as a change of the preferences of `id` in `slug` is sent; the check says it is still that page. */
  capture: (slug: string, id: string) => () => boolean;
  /** Whether the page's list was last read with the preferences of `id` as they are now. */
  listed: (id: string) => boolean;
  /** Reads the page's list again, with the pages it has. */
  relist: (slug: string, id: string) => Promise<unknown>;
};

/**
 * Follows a confirmed change of a list page's preferences in the same session and page: when it changed the list's
 * query, the list is read again with the pages it has; a refusal is said. Neither once the page is no longer the one
 * that sent it. A failed read of the list stays with the list, which shows it with its own retry.
 */
export function usePagePreferencesFollow({ capture, listed, relist }: PreferencesPage) {
  const refusal = useRefusalToast();
  const toast = useRef(refusal);
  toast.current = refusal;
  return useCallback(
    async (slug: string, id: string, change: () => Promise<void>) => {
      const current = capture(slug, id);
      await followInSession(change, {
        failed: (error) => {
          if (current()) toast.current(error);
        },
        done: async () => {
          if (!current() || listed(id)) return;
          // The binding retains read failures; the visible list owns its one refusal and GET-only retry.
          await followInSession(() => relist(slug, id), { failed: () => undefined });
        },
      });
    },
    [capture, listed, relist]
  );
}
