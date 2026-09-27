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

// The stores of a session act for it only (M2 design 7.1: a tab never writes as the wrong account). RootStore
// hands the client it is built or rebuilt with, bound to a session, to the stores that send requests with the
// session's token. The real stores, token manager and middleware, against a fake nerve.

// The page's localStorage: resetOnSignOut resets the theme there.
const page = new Map<string, string>();
vi.stubGlobal("localStorage", {
  getItem: (key: string) => page.get(key) ?? null,
  setItem: (key: string, value: string) => void page.set(key, value),
  removeItem: (key: string) => void page.delete(key),
});
// The stores switch the app's language; the translations are not what these tests look at.
vi.mock("@nerve/i18n", () => ({ FALLBACK_LANGUAGE: "en", setLanguage: async () => {} }));
// The stores get their session's client from RootStore; what else they import from api-client is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
// command-palette.store imports store-context, which builds the app's RootStore: a cycle through root.store.
vi.mock("@/lib/store-context", () => ({ store: {} }));

const { RootStore } = await import("@/store/root.store");

const REFRESH = "/api/v0/auth/refresh";
const ME = "/api/v0/me";
const PROFILE = "/api/v0/me/profile";
const X = "0123456789abcdef0123456789abcdef";
/** The session of another account, Y, which another tab signs in to. */
const Y = "fedcba9876543210fedcba9876543210";
const recordY = JSON.stringify({ refresh_token: "rt-y", login_id: Y });

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
  return { storage, nerve, tm, apiFor, followY };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("RootStore", () => {
  it("resetOnSignOut keeps the instance and the router, and starts the account's stores again", () => {
    const nerve = new FakeNerve();
    const store = new RootStore(nerve.client());
    const { instance, router, theme, user, workspaceRoot } = store;
    store.resetOnSignOut(nerve.client());
    expect(store.instance).toBe(instance);
    expect(store.router).toBe(router);
    expect(store.theme).toBe(theme);
    expect(store.user).not.toBe(user);
    expect(store.workspaceRoot).not.toBe(workspaceRoot);
  });

  it("sends as the session it is built for and, rebuilt for another session, as that one", async () => {
    const { nerve, apiFor, followY } = await setUp();
    const store = new RootStore(apiFor(X));
    const themed = track(store.user.userProfile.updateUserTheme("dark"));
    await until(() => nerve.calls.length === 1, "X's request");
    expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: PROFILE, authorization: "Bearer at-1" });
    nerve.calls[0]?.answer(json(200, { theme: "dark" }));
    await until(() => themed.settled, "X's answer");
    expect(themed.error).toBeUndefined();

    await followY();
    store.resetOnSignOut(apiFor(Y));
    const named = track(store.user.updateCurrentUser({ first_name: "Yvonne" }));
    await until(() => nerve.calls.length === 2, "Y's refresh");
    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: "rt-y" } });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "Y's request");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: ME, authorization: "Bearer at-2" });
    nerve.calls[2]?.answer(json(200, { first_name: "Yvonne" }));
    await until(() => named.settled, "Y's answer");
    expect(named.error).toBeUndefined();
    const stepped = track(store.user.userProfile.updateUserProfile({ onboarding_step: { profile_complete: true } }));
    await until(() => nerve.calls.length === 4, "Y's second request");
    expect(nerve.calls[3]).toMatchObject({ method: "PATCH", path: PROFILE, authorization: "Bearer at-2" });
    nerve.calls[3]?.answer(json(200, {}));
    await until(() => stepped.settled, "Y's second answer");
    expect(stepped.error).toBeUndefined();
  });

  it("stops the write a retired store makes from a stale reference, unsent: the profile step's save, then its step", async () => {
    const { storage, nerve, tm, apiFor, followY } = await setUp();
    const store = new RootStore(apiFor(X));
    // What the profile step holds from its render in X's session (onboarding/steps/profile/root.tsx saves the
    // names, then onboarding/root.tsx changes the step).
    const { updateCurrentUser } = store.user;
    const { updateUserProfile } = store.user.userProfile;
    const step = track(
      (async () => {
        await updateCurrentUser({ first_name: "Xavier" });
        await updateUserProfile({ onboarding_step: { profile_complete: true } });
      })()
    );
    await until(() => nerve.calls.length === 1, "X's save");
    expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: ME, authorization: "Bearer at-1" });
    // Another tab signs in as Y while the save is out; this tab follows, and its stores start again for Y.
    await followY();
    store.resetOnSignOut(apiFor(Y));
    // X's token is still good: the save succeeds, as X.
    nerve.calls[0]?.answer(json(200, { first_name: "Xavier" }));
    await until(() => step.settled || nerve.calls.length > 1, "the step, or another request");

    // The step change was X's: not sent with Y's token, nor as Y's refresh.
    expect(step.error).toBeInstanceOf(SessionChangedError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls.map((c) => `${c.method} ${c.path}`)).toEqual([`PATCH ${ME}`]);
    expect(tm.state).toEqual({ status: "signed-in", loginId: Y });
    expect(storage.data.get(AUTH_KEY)).toBe(recordY);
  });
});
