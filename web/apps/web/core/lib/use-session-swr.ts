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
 * The one SWR configuration of every session fetch, over the app's (WEB_SWR_CONFIG): a fetch is made as a page mounts
 * (the app's revalidateOnMount: true) and as its key changes (the app's revalidateIfStale: true, for a key SWR has an
 * answer of), not when the tab regains focus, and a refusal is not retried: the page shows it, with its own retry
 * where it has one. A list may shorten only the deduping interval: a restored permission needs a new read even
 * immediately after an invalidated one, while requests still in flight remain deduplicated. Capability owners may
 * disable SWR's reconnect read: their image lifetime refreshes on expiry, preserving the explicit retry budget. A route
 * each of whose visits makes its own read (the browse detail, which owns the detail host) disables SWR's mount read as
 * well and makes that read with mutate, so that an answer SWR keeps from an earlier visit is never taken for its own.
 */
const SESSION_FETCH: SWRConfiguration = { revalidateOnFocus: false, shouldRetryOnError: false };

/**
 * SWR for a fetch the tab's session makes into its stores (M3 design 7.1), the one way a component makes one: the
 * key is sessionKey's, so it carries the session's loginId; null fetches nothing, which is how a page leaves out
 * what its role may not read. The fetcher is given the fetch's arguments, every one known by then, so that what it
 * fetches is what the key names; it gives the store's Promise, so that a failure is the error SWR returns, never an
 * unhandled rejection. Every session fetch has the one configuration, SESSION_FETCH.
 */
export function useSessionSWR<T>(
  fetch: SessionFetch | null,
  fetcher: (...args: string[]) => Promise<T>,
  config?: Pick<SWRConfiguration, "dedupingInterval"> & { revalidateOnMount?: false; revalidateOnReconnect?: false }
): SWRResponse<T> {
  const session = useSession();
  return useSWR(fetch && sessionKey(session, ...fetch), ([, , ...args]: SessionKey) => fetcher(...args), {
    ...SESSION_FETCH,
    ...(config?.dedupingInterval === undefined ? {} : { dedupingInterval: config.dedupingInterval }),
    ...(config?.revalidateOnMount === false ? { revalidateOnMount: false } : {}),
    ...(config?.revalidateOnReconnect === false ? { revalidateOnReconnect: false } : {}),
  });
}
