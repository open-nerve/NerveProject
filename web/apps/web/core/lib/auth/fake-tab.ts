/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for api-client.ts, for the tests of what checks the tab's session (in-session.ts, and the pages that
// follow a change with it): a test file mocks that module with this one,
// vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab")), moves the tab from one session to another,
// and has nerve settle a change when it says (heldChange).

import { gate } from "./fake-browser";
import type { SessionState } from "./token-manager";

/** The token manager, as far as the session's check reads it: the tab's session. */
export const tokenManager: { state: SessionState } = { state: { status: "signed-in", loginId: "x" } };

/** Puts the tab in a session of its own, as a test starts: signed in, as login x. */
export function signedIn() {
  tokenManager.state = { status: "signed-in", loginId: "x" };
}

/** Another tab signs another account in, and this one follows it: a change sent before is the previous session's. */
export function switchAccount() {
  tokenManager.state = { status: "signed-in", loginId: `${tokenManager.state.loginId ?? ""}+` };
}

/** A change sent to nerve, and the two ways the test settles it: nerve answers it, or refuses it. */
export type HeldChange<T> = { sent: Promise<T>; answer: (value: T) => void; refuse: (error: unknown) => void };

/** A change nerve settles when the test says (a store's change, as a page's test mocks it): fake-browser.ts's gate. */
export function heldChange<T>(): HeldChange<T> {
  const { promise, open, fail } = gate<T>();
  return { sent: promise, answer: open, refuse: fail };
}

/** The ways a change sent before the tab switched settles after it, for it.each: answered, or refused. */
export const lateSettlings = [
  { settles: "is answered", settle: (change: HeldChange<undefined>) => change.answer(undefined) },
  { settles: "is refused", settle: (change: HeldChange<undefined>) => change.refuse(new Error("refused")) },
];

/**
 * Resolves once a page has followed a change that settled: what it does then runs in promises' steps, and a timer's
 * turn comes after them all.
 */
export function pageSettled(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}
