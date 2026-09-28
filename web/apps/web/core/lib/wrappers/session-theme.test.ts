/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { sessionTheme } from "@/lib/wrappers/session-theme";

// StoreWrapper sets the page's theme from what sessionTheme returns, each time its effect runs: when the session,
// its user store, the profile's theme or next-themes' setTheme changes. Two sessions, as StoreWrapper tells them
// apart: each has its own user store.
const X = { session: "X" };
const Y = { session: "Y" };

describe("sessionTheme", () => {
  it("is the default without a session, whatever the page took before", () => {
    expect(sessionTheme("signed-out", Y, undefined, X)).toEqual({ theme: "system", themedBy: undefined });
    expect(sessionTheme("signed-out", X, "dark", X)).toEqual({ theme: "system", themedBy: undefined });
  });

  it("is the profile's theme when a session's profile first arrives", () => {
    expect(sessionTheme("signed-in", X, "dark", undefined)).toEqual({ theme: "dark", themedBy: X });
  });

  it("keeps the theme shown until the session's profile arrives", () => {
    expect(sessionTheme("starting", X, undefined, undefined)).toEqual({ theme: undefined, themedBy: undefined });
    expect(sessionTheme("signed-in", Y, undefined, X)).toEqual({ theme: undefined, themedBy: X });
  });

  it("is set once per session: not when the effect runs again, nor when the profile's theme changes", () => {
    expect(sessionTheme("signed-in", X, "dark", X)).toEqual({ theme: undefined, themedBy: X });
    expect(sessionTheme("signed-in", X, "light", X)).toEqual({ theme: undefined, themedBy: X });
    expect(sessionTheme("unavailable", X, "dark", X)).toEqual({ theme: undefined, themedBy: X });
  });

  it("is a new session's profile's theme: a new user store is a new session", () => {
    expect(sessionTheme("signed-in", Y, "light", X)).toEqual({ theme: "light", themedBy: Y });
  });
});
