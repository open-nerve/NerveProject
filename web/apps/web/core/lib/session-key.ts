/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SessionState } from "@/lib/auth/token-manager";

/** The SWR key of a fetch of the tab's session: the fetch's name, the session's loginId, the fetch's arguments. */
export type SessionKey = readonly [name: string, loginId: string, ...args: string[]];

/**
 * The SWR key of a fetch the tab's session makes (M3 design 7.1), the one way one is written (useSessionSWR writes
 * them all): it carries the session's loginId, so that a new session, another account's or another sign-in's,
 * fetches again into its own stores instead of showing the answer of the session before. Null, which fetches
 * nothing, unless the tab is signed in and every argument is known.
 */
export function sessionKey(session: SessionState, name: string, ...args: (string | undefined)[]): SessionKey | null {
  const known = args.filter((arg): arg is string => arg !== undefined);
  if (session.status !== "signed-in" || session.loginId === undefined || known.length !== args.length) return null;
  return [name, session.loginId, ...known];
}
