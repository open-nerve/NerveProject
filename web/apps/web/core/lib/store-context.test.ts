/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { SharedStorage } from "@/lib/auth/fake-browser";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { GlobalViewStore } from "@/store/global-view.store";
import type { RootStore } from "@/store/root.store";

// Each session of the tab has its own RootStore (M2 design 7.1: a tab never writes as the wrong account):
// store-context.tsx builds one for the session the app loaded with, and a new one, with a client bound to the
// new session, each time the tab's session changes; the stores of a session reach their siblings through
// their own RootStore. The real store-context, stores, token manager and middleware, against a fake nerve:
// each test loads the app's modules afresh, with api-client.ts's composition on the fakes.

const AUTH_KEY = "nerve.auth";
const REFRESH = "/api/v0/auth/refresh";
const LOGOUT = "/api/v0/auth/logout";
const ME = "/api/v0/me";
const PROFILE = "/api/v0/me/profile";
const TOKENS = "/api/v0/me/api-tokens";
const X = "0123456789abcdef0123456789abcdef";
/** The sessions of other accounts, which other tabs sign in to. */
const Y = "fedcba9876543210fedcba9876543210";
const W = "cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd";
/** The session this tab signs in to itself. */
const Z = "abababababababababababababababab";
/** The record a sign-in as loginId writes. */
const record = (loginId: string) => JSON.stringify({ refresh_token: `rt-${loginId}`, login_id: loginId });
/** A personal access token of the account of loginId, as lists show it. */
const tokenOf = (loginId: string) => ({
  id: `${loginId.slice(0, 8)}-0000-4000-8000-000000000000`,
  label: `token of ${loginId}`,
  description: "",
  expired_at: null,
  last_used: null,
  created_at: "2026-09-27T00:00:00Z",
});

/** The page's theme and language, each time something set them. */
const page = vi.hoisted(() => ({ themes: [] as string[], languages: [] as string[] }));
vi.stubGlobal("localStorage", {
  getItem: () => null,
  setItem: (key: string, value: string) => {
    if (key === "theme") page.themes.push(value);
  },
  removeItem: () => {},
});
vi.mock("@nerve/i18n", () => ({
  FALLBACK_LANGUAGE: "en",
  setLanguage: async (language: string) => {
    page.languages.push(language);
  },
}));

/** Loads the app's modules afresh in a tab of a browser that holds X's record: its first refresh is out. */
async function load() {
  vi.resetModules();
  page.themes = [];
  page.languages = [];
  const storage = new SharedStorage();
  storage.data.set(AUTH_KEY, record(X));
  const nerve = new FakeNerve();
  /** The session of each client apiFor built: one for each RootStore. */
  const clients: (string | undefined)[] = [];
  // api-client.ts on the fakes: the token manager starts as the module loads, before the stores exist, and
  // apiFor puts the real middleware on each client.
  vi.doMock("@/lib/auth/api-client", async () => {
    const { RecordingLock } = await import("@/lib/auth/fake-browser");
    const { authMiddleware } = await import("@/lib/auth/auth-middleware");
    const { TokenManager } = await import("@/lib/auth/token-manager");
    const view = storage.tab("A");
    const tokenManager = new TokenManager({
      storage: view,
      lock: new RecordingLock(),
      client: nerve.client(),
      now: () => Date.now(),
      randomHex: () => Z,
    });
    view.onStorage((key) => {
      if (key === AUTH_KEY) tokenManager.handleStorageChange();
    });
    void tokenManager.start();
    return {
      publicClient: nerve.client(),
      tokenManager,
      apiFor: (loginId: string | undefined) => {
        clients.push(loginId);
        const api = nerve.client();
        api.use(authMiddleware(tokenManager, loginId));
        return api;
      },
    };
  });
  const { tokenManager: tm } = await import("@/lib/auth/api-client");
  const context = await import("@/lib/store-context");
  const { SessionChangedError } = await import("@/lib/auth/token-manager");
  /** Answers the first refresh: the tab is signed in as X. */
  const signedIn = async () => {
    await until(() => nerve.calls.length === 1, "the first refresh");
    nerve.calls[0]?.answer(json(200, nerve.tokens()));
    await until(() => tm.state.status === "signed-in", "X's session");
    nerve.calls.length = 0;
  };
  /** Another tab signs in as loginId, and this tab follows it. */
  const follow = async (loginId: string) => {
    storage.write(AUTH_KEY, record(loginId));
    await until(() => tm.state.loginId === loginId, `the switch to ${loginId}`);
  };
  return { storage, nerve, clients, tm, context, SessionChangedError, signedIn, follow };
}

// The first import of the stores compiles a few hundred modules: a second here, several on a busy CI runner,
// where it once ran over a test's 5 s. It happens once, before the tests and with a deadline of its own, and
// runs no app code (no token manager, no RootStore); each test's load() then evaluates the compiled modules
// afresh, in milliseconds.
beforeAll(async () => {
  vi.doMock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {}, apiFor: () => ({}) }));
  vi.doMock("@/lib/store-context", () => ({ rootStore: {} }));
  await import("@/store/root.store");
  vi.doUnmock("@/lib/store-context");
  vi.doUnmock("@/lib/auth/api-client");
}, 60_000);
beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("store-context", () => {
  it("starts a new RootStore for each new session, never within one, and tells its listeners", async () => {
    const { nerve, clients, tm, context, follow } = await load();
    const x = context.rootStore;
    expect(context.snapshot()).toBe(x);
    // The app loaded while the session X was starting.
    expect(clients).toEqual([X]);
    const started: RootStore[] = [];
    context.subscribe(() => started.push(context.snapshot()));
    // A component's subscription to the session comes after store-context's: at each change it hears of, the
    // RootStore of the tab's session is in place.
    const heard: [string | undefined, RootStore][] = [];
    tm.subscribe(() => heard.push([tm.state.loginId, context.rootStore]));

    // Within X: the first refresh fails for a passing reason, and the retry succeeds.
    await until(() => nerve.calls.length === 1, "the first refresh");
    nerve.calls[0]?.answer(problem(503, "server_busy", { "Retry-After": "1" }));
    await until(() => tm.state.status === "unavailable", "unavailable");
    await until(() => nerve.calls.length === 2, "the retry");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => tm.state.status === "signed-in", "X's session");
    expect(started).toEqual([]);
    expect(context.rootStore).toBe(x);

    // Another tab signs in as Y; this tab signs out; it signs in as Z; another tab signs in as W.
    await follow(Y);
    const out = track(tm.signOut());
    await until(() => nerve.calls.length === 3, "the logout");
    expect(nerve.calls[2]?.path).toBe(LOGOUT);
    nerve.calls[2]?.answer(noContent());
    await until(() => out.settled, "the sign-out");
    await tm.signIn(nerve.tokens());
    await follow(W);

    expect(clients).toEqual([X, Y, undefined, Z, W]);
    expect(started).toHaveLength(4);
    expect(new Set([x, ...started]).size).toBe(5);
    expect(context.rootStore).toBe(started[3]);
    expect(context.snapshot()).toBe(started[3]);
    const rootOf = new Map([X, Y, undefined, Z, W].map((loginId, i) => [loginId, [x, ...started][i]]));
    expect(heard.length).toBeGreaterThan(4);
    for (const [loginId, root] of heard) expect(root).toBe(rootOf.get(loginId));
    // Each new session starts with the system's theme and the default language, until its profile sets them.
    expect(page.themes).toEqual(["system", "system", "system", "system"]);
    expect(page.languages).toEqual(["en", "en", "en", "en"]);
  });

  it("leaves a session's stores with their own siblings: a retired store's request through its root rejects unsent", async () => {
    const { nerve, context, SessionChangedError, signedIn, follow } = await load();
    await signedIn();
    const x = context.rootStore;
    // A store of X's that reaches its siblings through its RootStore, as the stores do after an await
    // (global-view.store.ts, updateGlobalView).
    const globalView = x.globalView as GlobalViewStore;
    await follow(Y);

    const named = track(globalView.rootStore.user.updateCurrentUser({ first_name: "Xavier" }));
    await until(() => named.settled || nerve.calls.length > 0, "the answer, or a request");
    // Not sent with Y's token, nor as Y's refresh.
    expect(named.error).toBeInstanceOf(SessionChangedError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toEqual([]);

    // The RootStore of Y's session sends as Y.
    const y = context.rootStore;
    expect(y).not.toBe(x);
    const saved = track(y.user.updateCurrentUser({ first_name: "Yvonne" }));
    await until(() => nerve.calls.length === 1, "Y's refresh");
    expect(nerve.calls[0]).toMatchObject({ path: REFRESH, body: { refresh_token: `rt-${Y}` } });
    nerve.calls[0]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 2, "Y's request");
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: ME, authorization: "Bearer at-2" });
    nerve.calls[1]?.answer(json(200, { first_name: "Yvonne" }));
    await until(() => saved.settled, "Y's answer");
    expect(saved.error).toBeUndefined();
  });

  it("stops the write a retired store makes after an await, unsent: the profile step's save, then its step", async () => {
    const { storage, nerve, tm, context, SessionChangedError, signedIn, follow } = await load();
    await signedIn();
    // What the profile step holds from its render in X's session (onboarding/steps/profile/root.tsx saves the
    // names, then onboarding/root.tsx changes the step).
    const { updateCurrentUser } = context.rootStore.user;
    const { updateUserProfile } = context.rootStore.user.userProfile;
    const step = track(
      (async () => {
        await updateCurrentUser({ first_name: "Xavier" });
        await updateUserProfile({ onboarding_step: { profile_complete: true } });
      })()
    );
    await until(() => nerve.calls.length === 1, "X's save");
    expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: ME, authorization: "Bearer at-1" });
    // Another tab signs in as Y while the save is out; this tab follows, with a RootStore for Y.
    await follow(Y);
    // X's token is still good: the save succeeds, as X.
    nerve.calls[0]?.answer(json(200, { first_name: "Xavier" }));
    await until(() => step.settled || nerve.calls.length > 1, "the step, or another request");

    // The step change was X's: not sent with Y's token, nor as Y's refresh.
    expect(step.error).toBeInstanceOf(SessionChangedError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls.map((c) => `${c.method} ${c.path}`)).toEqual([`PATCH ${ME}`]);
    expect(nerve.to(PROFILE)).toEqual([]);
    expect(tm.state).toEqual({ status: "signed-in", loginId: Y });
    expect(storage.data.get(AUTH_KEY)).toBe(record(Y));
  });

  it("gives each session the tokens of its own account: a load cut by the switch stops, the new list loads as the new account", async () => {
    const { nerve, context, SessionChangedError, signedIn, follow } = await load();
    await signedIn();
    const x = context.rootStore;
    // The api-tokens page of X's session loads X's list, a page at a time.
    const listedX = track(x.user.apiTokens.fetchTokens());
    await until(() => nerve.calls.length === 1, "X's first page");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: TOKENS, authorization: "Bearer at-1" });
    // Another tab signs in as Y while the page is out; this tab follows.
    await follow(Y);
    nerve.calls[0]?.answer(json(200, { data: [tokenOf(X)], next_cursor: "c-1" }));
    await until(() => listedX.settled || nerve.calls.length > 1, "X's list, or another request");

    // X's next page is not asked for, as X or as Y; the load gives up quietly and shows nothing.
    expect(listedX).toEqual({ settled: true, value: undefined });
    expect(x.user.apiTokens.tokens).toBeUndefined();
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);

    // Y's session has a store of its own, which shows nothing until it has Y's list, loaded as Y.
    const y = context.rootStore;
    expect(y.user.apiTokens).not.toBe(x.user.apiTokens);
    expect(y.user.apiTokens.tokens).toBeUndefined();
    const listedY = track(y.user.apiTokens.fetchTokens());
    await until(() => nerve.calls.length === 2, "Y's refresh");
    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: `rt-${Y}` } });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "Y's list");
    expect(nerve.calls[2]).toMatchObject({ method: "GET", path: TOKENS, authorization: "Bearer at-2" });
    // Y's first page: no cursor of X's list.
    expect(nerve.calls[2]?.query).toEqual({ limit: "100" });
    nerve.calls[2]?.answer(json(200, { data: [tokenOf(Y)], next_cursor: null }));
    await until(() => listedY.settled, "Y's list");
    expect(y.user.apiTokens.tokens).toEqual([tokenOf(Y)]);

    // A revocation from X's page, which the tab no longer shows, is not sent: not with Y's token.
    const revoked = track(x.user.apiTokens.revokeToken(tokenOf(X).id));
    await until(() => revoked.settled || nerve.calls.length > 3, "the answer, or a request");
    expect(revoked.error).toBeInstanceOf(SessionChangedError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
    expect(y.user.apiTokens.tokens).toEqual([tokenOf(Y)]);
  });

  it("leaves the page's language to the tab's session now: no store sets it, and a profile is only nerve's answer", async () => {
    const { nerve, context, signedIn, follow } = await load();
    await signedIn();
    const x = context.rootStore;
    // X's profile load and X's change of language are out when another tab signs in as Y; this tab follows.
    const loaded = track(x.user.userProfile.fetchUserProfile());
    await until(() => nerve.calls.length === 1, "X's load");
    const changed = track(x.user.userProfile.updateUserProfile({ language: "zh-CN" }));
    await until(() => nerve.calls.length === 2, "X's change");
    await follow(Y);
    // nerve answers both as X, whose token is still good: both succeed, into X's retired profile only.
    nerve.calls[0]?.answer(json(200, { language: "zh-CN" }));
    nerve.calls[1]?.answer(json(200, { language: "zh-CN" }));
    await until(() => loaded.settled && changed.settled, "X's answers");
    expect([loaded.error, changed.error]).toEqual([undefined, undefined]);
    expect(x.user.userProfile.data).toEqual({ language: "zh-CN" });
    // The page's language was set once, by the switch, to the default; X's late answers did not reach it.
    expect(page.languages).toEqual(["en"]);

    // Y's profile is nerve's answer and nothing else: a change out leaves it, a refused one too.
    const profile = context.rootStore.user.userProfile;
    const fetched = track(profile.fetchUserProfile());
    await until(() => nerve.calls.length === 3, "Y's refresh");
    nerve.calls[2]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 4, "Y's load");
    nerve.calls[3]?.answer(json(200, { language: "en" }));
    await until(() => fetched.settled, "Y's profile");
    const refused = track(profile.updateUserProfile({ language: "zh-CN" }));
    await until(() => nerve.calls.length === 5, "Y's change");
    expect(profile.data).toEqual({ language: "en" });
    nerve.calls[4]?.answer(problem(500, "internal_error"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeDefined();
    expect(profile.data).toEqual({ language: "en" });
    const accepted = track(profile.updateUserProfile({ language: "zh-CN" }));
    await until(() => nerve.calls.length === 6, "Y's next change");
    expect(profile.data).toEqual({ language: "en" });
    nerve.calls[5]?.answer(json(200, { language: "zh-CN" }));
    await until(() => accepted.settled, "the acceptance");
    expect(profile.data).toEqual({ language: "zh-CN" });
    // Still no store set the page's language: the page follows the profile of the tab's session (StoreWrapper).
    expect(page.languages).toEqual(["en"]);
  });

  it("gives the code that reads the stores outside the components the RootStore of the session now", async () => {
    const { context, signedIn, follow } = await load();
    await signedIn();
    await follow(Y);
    // The RootStore the app renders with; command-palette.store reads the shortcuts modal of the one in
    // store-context's rootStore.
    const y = context.snapshot();
    y.powerK.toggleShortcutsListModal(true);
    expect(y.commandPalette.isAnyModalOpen).toBe(true);
  });
});
