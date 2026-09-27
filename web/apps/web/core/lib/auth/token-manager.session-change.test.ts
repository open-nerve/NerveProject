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

// A refresh belongs to the session it was asked for (M2 design 7.1): a tab's request made as X whose
// refresh waits for the lock while another tab signs in as Y must not get Y's token, although the tab has
// followed Y by the time its refresh holds the lock. With both kinds of lock, as in
// token-manager.tabs.test.ts.

const REFRESH = "/api/v0/auth/refresh";
const X = "0000000000000000000000000000000a";
/** The login_id of the first sign-in in these tabs: their randomHex numbers the sign-ins. */
const Y = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1";

type Kind = "navigator.locks" | "the lease";

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe.each<Kind>(["navigator.locks", "the lease"])("with %s", (kind) => {
  it("gives a request made as X a token of X's session or none, never Y's", async () => {
    const storage = new SharedStorage();
    storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
    const nerve = new FakeNerve();
    // navigator.locks: one queue for the whole browser, as the browser keeps it.
    const shared = new RecordingLock();
    const locks = {
      request: ((_name: string, task: () => Promise<unknown>) => shared.run(task)) as LockManager["request"],
    };
    let logins = 0;
    const tab = (id: string) => {
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
              grantedAs.push(tm.state.loginId);
              return task();
            }),
        },
        client: nerve.client(),
        now: () => Date.now(),
        randomHex: (bytes) => `${++logins}`.padStart(bytes * 2, "b"),
      });
      view.onStorage((key, value) => {
        if (key === AUTH_KEY) tm.handleStorageChange(value);
      });
      return { tm, grantedAs };
    };

    const a = tab("A");
    const started = track(a.tm.start());
    await until(() => nerve.calls.length === 1, "A's first refresh");
    // A 20 s token: A refreshes before every request.
    nerve.calls[0]?.answer(json(200, nerve.tokens(20)));
    await until(() => started.settled, "A's start");
    a.grantedAs.length = 0;

    // Tab B signs in as Y, holding the lock; meanwhile a request of A, made as X, needs a refresh.
    const b = tab("B");
    const signedIn = track(b.tm.signIn({ ...nerve.tokens(), refresh_token: "rt-y" }));
    const token = track(a.tm.accessToken());
    await until(() => signedIn.settled && a.grantedAs.length === 1, "B's sign-in, then A's refresh at the lock");
    expect(b.tm.state).toEqual({ status: "signed-in", loginId: Y });
    // A heard of B's sign-in before its refresh got the lock: the case this test is about.
    expect(a.grantedAs).toEqual([Y]);

    // Had A refreshed B's record for the request, nerve would give it Y's tokens.
    await until(() => token.settled || nerve.calls.length === 2, "A's token or a refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => token.settled, "A's token");
    expect(token.error).toBeInstanceOf(SessionChangedError);
    expect(nerve.to(REFRESH)).toHaveLength(1);
    // A follows B's session, whose record stays as B wrote it.
    expect(a.tm.state).toEqual({ status: "signed-in", loginId: Y });
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({ refresh_token: "rt-y", login_id: Y });
  });
});
