/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it, vi } from "vitest";
import type { SessionState } from "@/lib/auth/token-manager";

// The tab's session as the token manager has it, which the test moves from one account to another.
const tab = vi.hoisted((): { state: SessionState } => ({ state: { status: "signed-in", loginId: "x" } }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: tab }));

const { sessionGuard } = await import("./in-session");

describe("sessionGuard", () => {
  it("holds while the tab stays in the session the change was sent in, and fails once it is in another", () => {
    tab.state = { status: "signed-in", loginId: "x" };
    const inSession = sessionGuard();
    expect(inSession()).toBe(true);
    // a refresh within the session, or a passing failure, keeps the session
    tab.state = { status: "unavailable", loginId: "x", retryAt: 1 };
    expect(inSession()).toBe(true);
    // another tab signs in as Y, and this one follows
    tab.state = { status: "signed-in", loginId: "y" };
    expect(inSession()).toBe(false);
    // the check is of the session the change was sent in: one taken now is Y's
    expect(sessionGuard()()).toBe(true);
  });

  it("fails once the tab signed out, and after a sign-in as the same account again", () => {
    tab.state = { status: "signed-in", loginId: "x" };
    const inSession = sessionGuard();
    tab.state = { status: "signed-out" };
    expect(inSession()).toBe(false);
    // a new sign-in is a new session: another loginId, though the account is the same
    tab.state = { status: "signed-in", loginId: "x2" };
    expect(inSession()).toBe(false);
  });
});
