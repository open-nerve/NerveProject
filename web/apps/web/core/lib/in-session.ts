/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { tokenManager } from "@/lib/auth/api-client";

/**
 * Takes the tab's session as a component sends a change, and gives the check of it: whether the tab is still in
 * that session (v0 design 7.7, M3 design 7.1). A change sent before another tab moved this one to another account
 * may still succeed afterwards, and the component that sent it still gets the answer: it follows the answer
 * (navigates, says so, changes the page) only while the check holds, since the page is then the other account's.
 */
export function sessionGuard(): () => boolean {
  const loginId = tokenManager.state.loginId;
  return () => tokenManager.state.loginId === loginId;
}

/**
 * A follower: what a page does once a change it sent has settled, done in its turn, or, when it goes on past it (a
 * navigation, the onboarding's end), returning what settles once it is done.
 */
type Follower<A> = ((settled: A) => void) | ((settled: A) => Promise<void>);

/**
 * What a page does once a change it sent has settled: with nerve's answer (nothing, for a change whose answer the
 * stores show), or with why it failed.
 */
export type ChangeFollowers<T> = { done?: Follower<T>; failed: Follower<unknown> };

/**
 * Sends a change, and follows how it settles on the page only while the tab is in the session the change was sent in
 * (sessionGuard): done with nerve's answer, failed with the error; neither once another tab has moved this one to
 * another account, whose page it then is, whatever the answer (M3 design 7.1). Settles once the change has and, when
 * the page follows it, once what the follower returned has too, so that a page busy until then stays busy while the
 * follow-up is under way. Never rejects: the change's failure is the followers' to show, and a follower's own failure
 * (it throws, or what it returned rejects) is a fault of the page's code, which goes to the browser's report of
 * uncaught errors (reportError: the console, and the window's error event), not to the caller.
 */
export async function followInSession<T>(change: () => Promise<T>, followers: ChangeFollowers<T>): Promise<void> {
  const inSession = sessionGuard();
  const outcome = await change().then(
    (answer) => ({ settled: "done" as const, answer }),
    (error: unknown) => ({ settled: "failed" as const, error })
  );
  if (!inSession()) return;
  try {
    if (outcome.settled === "done") await followers.done?.(outcome.answer);
    else await followers.failed(outcome.error);
  } catch (fault) {
    reportError(fault);
  }
}
