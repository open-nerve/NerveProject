/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SessionState } from "@/lib/auth/token-manager";

/**
 * The page's theme as the tab's session has it (v0 design 7.7): without a session, the default; with one, its
 * profile's, once per session, when the profile first arrives. session identifies the session (each has its own
 * user store) and themedBy the one whose profile's theme the page took last. It returns the theme to set, or
 * undefined to keep the one shown, and the session themedBy is now. Once per session, because a later change of
 * the profile's theme is applied by the component that makes it (theme-switcher.tsx), and a stale profile must not
 * set its theme again each time the page's theme changes (next-themes' setTheme changes with it, in this tab or
 * another).
 */
export function sessionTheme<S>(
  status: SessionState["status"],
  session: S,
  profileTheme: string | undefined,
  themedBy: S | undefined
): { theme: string | undefined; themedBy: S | undefined } {
  if (status === "signed-out") return { theme: "system", themedBy: undefined };
  if (!profileTheme || themedBy === session) return { theme: undefined, themedBy };
  return { theme: profileTheme, themedBy: session };
}
