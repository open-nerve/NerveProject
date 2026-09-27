/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it, vi } from "vitest";

// The stores serve one session at a time (M2 design 7.1): store-context.tsx builds them for the session the
// app loaded with, and again, with a client bound to the new session, whenever the tab's session changes.

type State = { status: string; loginId?: string; retryAt?: number };

const fake = vi.hoisted(() => {
  const listeners = new Set<() => void>();
  const tokenManager = {
    state: { status: "starting", loginId: "X" } as State,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
  /** A change of the token manager's state, which it tells its listeners of. */
  const set = (state: State) => {
    tokenManager.state = state;
    for (const listener of listeners) listener();
  };
  /** The client of each generation of stores: RootStore's first, then each rebuild's. */
  const generations: { loginId: string | undefined }[] = [];
  return { tokenManager, set, generations };
});

/** A new client for each call, which says the session it is bound to. */
function apiFor(loginId: string | undefined) {
  return { loginId };
}

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: fake.tokenManager, apiFor }));
vi.mock("@/store/root.store", () => ({
  RootStore: class {
    constructor(api: { loginId: string | undefined }) {
      fake.generations.push(api);
    }
    resetOnSignOut(api: { loginId: string | undefined }) {
      fake.generations.push(api);
    }
  },
}));

/** The session of each generation of stores so far. */
const sessions = () => fake.generations.map((api) => api.loginId);

describe("store-context", () => {
  it("builds the stores for the session the app loaded with, and again for each new session only", async () => {
    await import("@/lib/store-context");
    // The app loaded while the session X was starting.
    expect(sessions()).toEqual(["X"]);
    // Its first refresh, a redundant notification, a passing failure and the retry: the same session.
    fake.set({ status: "signed-in", loginId: "X" });
    fake.set({ status: "signed-in", loginId: "X" });
    fake.set({ status: "unavailable", loginId: "X", retryAt: 1 });
    fake.set({ status: "signed-in", loginId: "X" });
    expect(sessions()).toEqual(["X"]);
    // Another tab signed in as another account.
    fake.set({ status: "signed-in", loginId: "Y" });
    fake.set({ status: "signed-in", loginId: "Y" });
    expect(sessions()).toEqual(["X", "Y"]);
    // Signed out; then a sign-in, and another tab's sign-in: stores for no session, then for each new one.
    fake.set({ status: "signed-out" });
    expect(sessions()).toEqual(["X", "Y", undefined]);
    fake.set({ status: "signed-in", loginId: "Z" });
    fake.set({ status: "signed-in", loginId: "W" });
    expect(sessions()).toEqual(["X", "Y", undefined, "Z", "W"]);
  });
});
