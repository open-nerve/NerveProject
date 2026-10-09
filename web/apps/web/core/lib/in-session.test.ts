/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  heldChange,
  lateSettlings,
  signedIn,
  switchAccount,
  tokenManager as tab,
  type HeldChange,
} from "@/lib/auth/fake-tab";

// The tab's session as the token manager has it, which the test moves from one account to another (fake-tab.ts).
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

const { followInSession, sessionGuard } = await import("./in-session");

beforeEach(() => {
  signedIn();
});

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

describe("followInSession", () => {
  /** Follows a change nerve settles when the test says (heldChange): what the page did once it settled. */
  function follow<T>(change: HeldChange<T>) {
    const followed: string[] = [];
    const following = followInSession(() => change.sent, {
      done: (answer) => followed.push(`done: ${String(answer)}`),
      failed: (error) => followed.push(`failed: ${String(error)}`),
    });
    return { followed, following };
  }

  it("follows nerve's answer, and its refusal, while the tab is in the session the change was sent in", async () => {
    const answered = heldChange<string>();
    const afterAnswer = follow(answered);
    answered.answer("acme");
    await afterAnswer.following;
    const refused = heldChange<string>();
    const afterRefusal = follow(refused);
    refused.refuse("refused");
    await afterRefusal.following;
    expect([afterAnswer.followed, afterRefusal.followed]).toEqual([["done: acme"], ["failed: refused"]]);
  });

  it.each(lateSettlings)(
    "follows nothing when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      const after = follow(change);
      switchAccount();
      settle(change);
      await after.following;
      expect(after.followed).toEqual([]);
    }
  );

  it("takes the session as the change is sent: a change sent after the switch is followed", async () => {
    switchAccount();
    const change = heldChange<string>();
    const after = follow(change);
    change.answer("acme");
    await after.following;
    expect(after.followed).toEqual(["done: acme"]);
  });
});
