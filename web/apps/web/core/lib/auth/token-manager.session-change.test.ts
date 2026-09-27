/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json } from "./fake-nerve";
import { track, until } from "./fake-time";
import { leaseLock, webLock } from "./refresh-lock";
import { AUTH_KEY, SessionChangedError, TokenManager } from "./token-manager";

// An operation that changes the session acts only on the session it was asked for (M2 design 7.1): a
// refresh, a sign-out or the end of a session asked for as X, whose turn at the lock comes after another
// tab signed in as Y, never refreshes, logs out or removes Y's session, although the tab has followed Y by
// the time it holds the lock. With both kinds of lock, as in token-manager.tabs.test.ts; the storage events
// are held back and reach the tabs as A's task gets the lock, so the order does not rest on the fake's.

const REFRESH = "/api/v0/auth/refresh";
const LOGOUT = "/api/v0/auth/logout";
const X = "0000000000000000000000000000000a";
/** The login_id of the first sign-in in these tabs: their randomHex numbers the sign-ins. */
const Y = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1";

type Kind = "navigator.locks" | "the lease";

/** One browser with the record of X, whose tabs record their session each time the lock is granted. */
function browser(kind: Kind) {
  const storage = new SharedStorage();
  storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
  const nerve = new FakeNerve();
  // navigator.locks: one queue for the whole browser, as the browser keeps it.
  const shared = new RecordingLock();
  const locks = {
    request: ((_name: string, task: () => Promise<unknown>) => shared.run(task)) as LockManager["request"],
  };
  let logins = 0;
  /** A tab; atGrant runs each time one of its tasks gets the lock, before the task. */
  const tab = (id: string, atGrant?: () => void) => {
    const view = storage.tab(id);
    const lock =
      kind === "navigator.locks"
        ? webLock(locks)
        : leaseLock({ storage: view, onStorage: view.onStorage, now: () => Date.now(), tabId: id });
    // The tab's session each time the lock is granted to one of its tasks.
    const grantedAs: (string | undefined)[] = [];
    const tm: TokenManager = new TokenManager({
      storage: view,
      lock: {
        run: (task) =>
          lock.run(() => {
            atGrant?.();
            grantedAs.push(tm.state.loginId);
            return task();
          }),
      },
      client: nerve.client(),
      now: () => Date.now(),
      randomHex: (bytes) => `${++logins}`.padStart(bytes * 2, "b"),
    });
    view.onStorage((key) => {
      if (key === AUTH_KEY) tm.handleStorageChange();
    });
    return { tm, grantedAs };
  };
  /**
   * Tab A, started as X with a 20 s token: it refreshes before every request. Each time one of its tasks
   * gets the lock, the events held back (storage.hold()) reach the tabs first: A has heard of B's sign-in
   * by then, one order a browser may give, forced here.
   */
  const tabA = async () => {
    const a = tab("A", () => storage.deliver());
    const started = track(a.tm.start());
    await until(() => nerve.calls.length === 1, "A's first refresh");
    nerve.calls[0]?.answer(json(200, nerve.tokens(20)));
    await until(() => started.settled, "A's start");
    a.grantedAs.length = 0;
    return a;
  };
  const stored = () => JSON.parse(storage.data.get(AUTH_KEY) ?? "null") as unknown;
  return { storage, nerve, tab, tabA, stored };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe.each<Kind>(["navigator.locks", "the lease"])("with %s", (kind) => {
  it("gives a request made as X a token of X's session or none, never Y's", async () => {
    const { storage, nerve, tab, tabA, stored } = browser(kind);
    const a = await tabA();

    // Tab B signs in as Y, holding the lock; meanwhile a request of A, made as X, needs a refresh.
    storage.hold();
    const b = tab("B");
    const signedIn = track(b.tm.signIn({ ...nerve.tokens(), refresh_token: "rt-y" }));
    const token = track(a.tm.accessToken());
    await until(() => signedIn.settled && a.grantedAs.length === 1, "B's sign-in, then A's refresh at the lock");
    expect(b.tm.state).toEqual({ status: "signed-in", loginId: Y });
    // A heard of B's sign-in as its refresh got the lock: the case this test is about.
    expect(a.grantedAs).toEqual([Y]);

    // Had A refreshed B's record for the request, nerve would give it Y's tokens.
    await until(() => token.settled || nerve.calls.length === 2, "A's token or a refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => token.settled, "A's token");
    expect(token.error).toBeInstanceOf(SessionChangedError);
    expect(nerve.to(REFRESH)).toHaveLength(1);
    // A follows B's session, whose record stays as B wrote it.
    expect(a.tm.state).toEqual({ status: "signed-in", loginId: Y });
    expect(stored()).toEqual({ refresh_token: "rt-y", login_id: Y });
  });

  it("logs nobody out when a sign-out asked for as X gets the lock after another tab's sign-in as Y", async () => {
    const { storage, nerve, tab, tabA, stored } = browser(kind);
    const a = await tabA();

    // Tab B's sign-in as Y takes the lock ahead of A's sign-out, asked for as X.
    storage.hold();
    const b = tab("B");
    const signedIn = track(b.tm.signIn({ ...nerve.tokens(), refresh_token: "rt-y" }));
    const out = track(a.tm.signOut());
    expect(a.tm.state.loginId).toBe(X);
    await until(() => signedIn.settled && out.settled, "B's sign-in and A's sign-out");
    // A heard of B's sign-in as its sign-out got the lock: the case this test is about.
    expect(a.grantedAs).toEqual([Y]);

    // B's sign-in replaced X's refresh token, so nothing of X is left to log out: A follows Y.
    expect(out.error).toBeUndefined();
    expect(nerve.to(LOGOUT)).toEqual([]);
    expect(stored()).toEqual({ refresh_token: "rt-y", login_id: Y });
    expect([a.tm.state, b.tm.state]).toEqual([
      { status: "signed-in", loginId: Y },
      { status: "signed-in", loginId: Y },
    ]);
  });

  it("keeps Y's session when a request of X, refused again, ends X's after another tab's sign-in as Y", async () => {
    const { storage, nerve, tab, tabA, stored } = browser(kind);
    const a = await tabA();

    // A request of A, made as X, was refused again after its replay, so A ends X's session; tab B's
    // sign-in as Y takes the lock first.
    storage.hold();
    const b = tab("B");
    const signedIn = track(b.tm.signIn({ ...nerve.tokens(), refresh_token: "rt-y" }));
    const ended = track(a.tm.endSession(X));
    expect(a.tm.state.loginId).toBe(X);
    await until(() => signedIn.settled && ended.settled, "B's sign-in and the end of X's session");
    // A heard of B's sign-in, and followed Y, as its end of X's session got the lock: the case this test
    // is about.
    expect(a.grantedAs).toEqual([Y]);

    expect(ended.error).toBeUndefined();
    expect(ended.value).toBe(false);
    expect(stored()).toEqual({ refresh_token: "rt-y", login_id: Y });
    expect([a.tm.state, b.tm.state]).toEqual([
      { status: "signed-in", loginId: Y },
      { status: "signed-in", loginId: Y },
    ]);
    expect(nerve.calls).toHaveLength(1);
  });
});
