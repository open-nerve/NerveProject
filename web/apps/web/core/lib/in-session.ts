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
 * What a page does once a change it sent has settled: with nerve's answer (nothing, for a change whose answer the
 * stores show), or with why it failed.
 */
export type ChangeFollowers<T> = { done?: (answer: T) => void; failed: (error: unknown) => void };

/**
 * Sends a change, and follows how it settles on the page only while the tab is in the session the change was sent in
 * (sessionGuard): done with nerve's answer, failed with the error; neither once another tab has moved this one to
 * another account, whose page it then is, whatever the answer (M3 design 7.1). Settles once the change has, and never
 * rejects: its failure is the followers' to show.
 */
export async function followInSession<T>(change: () => Promise<T>, followers: ChangeFollowers<T>): Promise<void> {
  const inSession = sessionGuard();
  const outcome = await change().then(
    (answer) => ({ settled: "done" as const, answer }),
    (error: unknown) => ({ settled: "failed" as const, error })
  );
  if (!inSession()) return;
  if (outcome.settled === "done") followers.done?.(outcome.answer);
  else followers.failed(outcome.error);
}
