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
