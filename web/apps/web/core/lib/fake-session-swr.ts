/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for use-session-swr.ts, for the tests of the hooks that make a page's fetches (M3 design 7.1): a test
// file mocks that module with this one, vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr")), and
// reads what its hook handed over. The hooks then run as plain functions, outside React. How a fetch is keyed is
// use-session-swr.test.ts.

import type { SessionFetch } from "./use-session-swr";

/** What a hook handed useSessionSWR: the fetch, the fetcher of its arguments, and SWR's configuration, if any. */
export type HandedFetch = [
  fetch: SessionFetch | null,
  fetcher: (...args: string[]) => Promise<unknown>,
  config?: unknown,
];

/** Every fetch handed over so far, in order; a test empties it before each case. */
export const handed: HandedFetch[] = [];

/** What each call gives back, as SWR would (by default, nothing yet); a test that sets it resets it before each case. */
export const response: { current: object } = { current: {} };

export function useSessionSWR(...call: HandedFetch): object {
  handed.push(call);
  return response.current;
}

/** Calls each fetcher handed over with its fetch's arguments, as SWR calls it with the key's; null fetches nothing. */
export function fetchHanded(): Promise<unknown[]> {
  return Promise.all(handed.flatMap(([fetch, fetcher]) => (fetch ? [fetcher(...known(fetch.slice(1)))] : [])));
}

const known = (args: (string | undefined)[]) => args.filter((arg) => arg !== undefined);
