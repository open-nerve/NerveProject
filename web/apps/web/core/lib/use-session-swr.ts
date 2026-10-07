/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import useSWR from "swr";
import type { SWRConfiguration, SWRResponse } from "swr";
import { useSession } from "@/lib/auth/use-session";
import { sessionKey } from "@/lib/session-key";
import type { SessionKey } from "@/lib/session-key";

/** A fetch of the tab's session: its name and its arguments, of which an unknown one fetches nothing yet. */
export type SessionFetch = readonly [name: string, ...args: (string | undefined)[]];

/**
 * SWR for a fetch the tab's session makes into its stores (M3 design 7.1), the one way a component makes one: the
 * key is sessionKey's, so it carries the session's loginId; null fetches nothing, which is how a page leaves out
 * what its role may not read. The fetcher is given the fetch's arguments, every one known by then, so that what it
 * fetches is what the key names; it gives the store's Promise, so that a failure is the error SWR returns, never an
 * unhandled rejection.
 */
export function useSessionSWR<T>(
  fetch: SessionFetch | null,
  fetcher: (...args: string[]) => Promise<T>,
  config?: SWRConfiguration<T>
): SWRResponse<T> {
  const session = useSession();
  return useSWR(fetch && sessionKey(session, ...fetch), ([, , ...args]: SessionKey) => fetcher(...args), config);
}
