/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiClient } from "@nerve/api-client";
import { authMiddleware } from "@/lib/auth/auth-middleware";
import { RecordingLock, SharedStorage } from "@/lib/auth/fake-browser";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { AUTH_KEY, SessionChangedError, TokenManager } from "@/lib/auth/token-manager";
import type { GlobalViewStore } from "@/store/global-view.store";
import type { ProfileStore } from "@/store/user/profile.store";

// A RootStore holds the stores of one session (M2 design 7.1: a tab never writes as the wrong account): it hands
// the client it is built with, bound to that session, to the stores that send requests with the session's
// token, and builds its stores anew for each session but for what is the page's. The real stores, token
// manager and middleware, against a fake nerve. How store-context.tsx starts a RootStore for each session of
// the tab is store-context.test.ts.

// The page's localStorage, which stores read as they are built.
const page = new Map<string, string>();
vi.stubGlobal("localStorage", {
  getItem: (key: string) => page.get(key) ?? null,
  setItem: (key: string, value: string) => void page.set(key, value),
  removeItem: (key: string) => void page.delete(key),
});
// The stores switch the app's language; the translations are not what these tests look at.
vi.mock("@nerve/i18n", () => ({ setLanguage: async () => {} }));
// The stores get their session's client from RootStore; what else they import from api-client is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
// command-palette.store imports store-context, which builds the app's RootStore: a cycle through root.store.
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

const { RootStore } = await import("@/store/root.store");

const REFRESH = "/api/v0/auth/refresh";
const ME = "/api/v0/me";
const PROFILE = "/api/v0/me/profile";
const X = "0123456789abcdef0123456789abcdef";
/** The session of another account, Y, which another tab signs in to. */
const Y = "fedcba9876543210fedcba9876543210";
const recordY = JSON.stringify({ refresh_token: "rt-y", login_id: Y });
/** The stores that are the page's, not the account's. */
const PAGE_STORES = ["instance", "router", "theme"];

/** A tab in the session X; apiFor builds a client for the stores of a session, as api-client.ts does. */
async function setUp() {
  const storage = new SharedStorage();
  storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
  const nerve = new FakeNerve();
  const view = storage.tab("A");
  const tm = new TokenManager({
    storage: view,
    lock: new RecordingLock(),
    client: nerve.client(),
    now: () => Date.now(),
    randomHex: () => X,
  });
  view.onStorage((key) => {
    if (key === AUTH_KEY) tm.handleStorageChange();
  });
  const started = track(tm.start());
  await until(() => nerve.calls.length === 1, "the first refresh");
  nerve.calls[0]?.answer(json(200, nerve.tokens()));
  await until(() => started.settled, "the start");
  nerve.calls.length = 0;
  const apiFor = (loginId: string | undefined): ApiClient => {
    const api = nerve.client();
    api.use(authMiddleware(tm, loginId));
    return api;
  };
  /** Another tab signs in as Y, and this tab follows it. */
  const followY = async () => {
    storage.write(AUTH_KEY, recordY);
    await until(() => tm.state.loginId === Y, "the switch to Y");
  };
  return { nerve, apiFor, followY };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("RootStore", () => {
  it("built for the next session, goes on with the page's stores and builds the account's anew", () => {
    const nerve = new FakeNerve();
    const x = new RootStore(nerve.client());
    const y = new RootStore(nerve.client(), x);
    const stores = Object.keys(x) as (keyof typeof x)[];
    expect(stores).toHaveLength(23);
    for (const name of stores) {
      if (PAGE_STORES.includes(name)) expect(y[name], name).toBe(x[name]);
      else expect(y[name], name).not.toBe(x[name]);
    }
    // The page's stores hold nothing of the RootStore before; the new stores reach their siblings through y.
    for (const name of PAGE_STORES) expect(Object.values(x[name as keyof typeof x])).not.toContain(x);
    expect((y.user.userProfile as ProfileStore).store).toBe(y);
    expect((y.globalView as GlobalViewStore).rootStore).toBe(y);
    expect(y.issue.rootStore).toBe(y);
  });

  it("sends as the session it is built for: X's as X, and the one built for Y as Y", async () => {
    const { nerve, apiFor, followY } = await setUp();
    const x = new RootStore(apiFor(X));
    const themed = track(x.user.userProfile.updateUserTheme("dark"));
    await until(() => nerve.calls.length === 1, "X's request");
    expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: PROFILE, authorization: "Bearer at-1" });
    nerve.calls[0]?.answer(json(200, { theme: "dark" }));
    await until(() => themed.settled, "X's answer");
    expect(themed.error).toBeUndefined();

    await followY();
    const y = new RootStore(apiFor(Y), x);
    const named = track(y.user.updateCurrentUser({ first_name: "Yvonne" }));
    await until(() => nerve.calls.length === 2, "Y's refresh");
    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: "rt-y" } });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "Y's request");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: ME, authorization: "Bearer at-2" });
    nerve.calls[2]?.answer(json(200, { first_name: "Yvonne" }));
    await until(() => named.settled, "Y's answer");
    expect(named.error).toBeUndefined();

    // X's stores send nothing now: their client is bound to X.
    const stepped = track(x.user.userProfile.updateUserProfile({ onboarding_step: { profile_complete: true } }));
    await until(() => stepped.settled || nerve.calls.length > 3, "X's answer, or a request");
    expect(stepped.error).toBeInstanceOf(SessionChangedError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });
});
