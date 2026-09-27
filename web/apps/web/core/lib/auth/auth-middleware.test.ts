/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authMiddleware } from "./auth-middleware";
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json, problem } from "./fake-nerve";
import { track, until } from "./fake-time";
import { AUTH_KEY, SessionChangedError, SessionUnavailableError, TokenManager } from "./token-manager";

// The client the app uses, with the middleware on a real token manager, against a fake nerve that answers
// when the test says (M2 design 7.1, 9.4).

const REFRESH = "/api/v0/auth/refresh";
const ME = "/api/v0/me";
const PROFILE = "/api/v0/me/profile";
const loginId = "0123456789abcdef0123456789abcdef";
/** The session of another account, Y, which another tab signs in to. */
const loginIdY = "fedcba9876543210fedcba9876543210";
const recordY = JSON.stringify({ refresh_token: "rt-y", login_id: loginIdY });
const user = {
  id: "5f0c1b1e-8a6d-4d0e-9d0b-0a8f5f5f5f5f",
  email: "ada@example.com",
  first_name: "Ada",
  last_name: "",
  display_name: "ada",
  user_timezone: "UTC",
  avatar_url: null,
  cover_image_url: null,
  created_at: "2026-09-27T00:00:00Z",
};

async function setUp(record = true) {
  const storage = new SharedStorage();
  if (record) storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: loginId }));
  const nerve = new FakeNerve();
  const view = storage.tab("A");
  const tm = new TokenManager({
    storage: view,
    lock: new RecordingLock(),
    client: nerve.client(),
    now: () => Date.now(),
    randomHex: () => loginId,
  });
  // The tab hears the other tabs' writes of the record, as the app wires it.
  view.onStorage((key) => {
    if (key === AUTH_KEY) tm.handleStorageChange();
  });
  const started = track(tm.start());
  if (record) {
    await until(() => nerve.calls.length === 1, "the first refresh");
    nerve.calls[0]?.answer(json(200, nerve.tokens()));
  }
  await until(() => started.settled, "the start");
  nerve.calls.length = 0;
  // The client of the stores of the tab's session, as the app builds one for them.
  const api = nerve.client();
  api.use(authMiddleware(tm, tm.state.loginId));
  return { storage, nerve, tm, api };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("authMiddleware", () => {
  it("sends the access token", async () => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: ME, authorization: "Bearer at-1" });
    nerve.calls[0]?.answer(json(200, user));
    await until(() => me.settled, "the answer");
    expect(me.value?.data).toEqual(user);
  });

  it("sends no token without a session, and hands a 401 back as it is", async () => {
    const { nerve, api, tm } = await setUp(false);
    expect(tm.state.status).toBe("signed-out");
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBeNull();
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(401);
    expect(me.value?.error).toMatchObject({ code: "unauthorized" });
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });

  it("does not send a request made without a session again when a session began before its 401", async () => {
    const { nerve, api, tm } = await setUp(false);
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBeNull();
    // A sign-in while the request is out: the request stays the signed-out one it was.
    await tm.signIn(nerve.tokens());
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(401);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });

  it("renews the token after a 401 and sends the same request again, once", async () => {
    const { nerve, api } = await setUp();
    const saved = track(api.PATCH(PROFILE, { body: { theme: "dark" } }));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: "rt-1" }, authorization: null });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    expect(nerve.calls[2]).toMatchObject({
      method: "PATCH",
      path: PROFILE,
      body: { theme: "dark" },
      authorization: "Bearer at-2",
    });
    nerve.calls[2]?.answer(json(200, { theme: "dark" }));
    await until(() => saved.settled, "the answer");
    expect(saved.value?.data).toEqual({ theme: "dark" });
    expect(nerve.calls).toHaveLength(3);
  });

  it("renews once for requests refused at the same time", async () => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    const saved = track(api.PATCH(PROFILE, { body: { language: "en" } }));
    await until(() => nerve.calls.length === 2, "the requests");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    nerve.calls[1]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.to(REFRESH).length === 1, "the refresh");
    await vi.advanceTimersByTimeAsync(100);
    expect(nerve.to(REFRESH)).toHaveLength(1);
    nerve.to(REFRESH)[0]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 5, "the requests again");
    const again = nerve.calls.slice(3);
    expect(again.map((c) => [c.method, c.authorization])).toEqual([
      ["GET", "Bearer at-2"],
      ["PATCH", "Bearer at-2"],
    ]);
    again[0]?.answer(json(200, user));
    again[1]?.answer(json(200, { language: "en" }));
    await until(() => me.settled && saved.settled, "the answers");
    expect([me.value?.response.status, saved.value?.response.status]).toEqual([200, 200]);
  });

  it("ends the session when the refresh answers 401, and hands the first 401 back", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(problem(401, "identity.refresh_token_invalid"));
    await until(() => me.settled, "the answer");

    expect(me.value?.response.status).toBe(401);
    expect(me.value?.error).toMatchObject({ code: "unauthorized" });
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    expect(nerve.calls).toHaveLength(2);
  });

  it("ends the session when the request sent again is refused too", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    nerve.calls[2]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");

    expect(me.value?.response.status).toBe(401);
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });

  it("stops a request with SessionChangedError when the tab has moved to another session by its 401", async () => {
    const { storage, nerve, api, tm } = await setUp();
    const saved = track(api.PATCH(ME, { body: { first_name: "Xavier" } }));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBe("Bearer at-1");
    // Another tab signs in as Y while the request is out; this tab follows.
    storage.write(AUTH_KEY, recordY);
    await until(() => tm.state.loginId === loginIdY, "the switch to Y");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => saved.settled || nerve.calls.length > 1, "the answer, or another request");

    // Neither renewed in Y's session nor sent again with Y's token, nor handed back as a 401 in Y's session.
    expect(nerve.calls.map((c) => `${c.method} ${c.path}`)).toEqual([`PATCH ${ME}`]);
    expect(saved.error).toBeInstanceOf(SessionChangedError);
    expect(tm.state).toEqual({ status: "signed-in", loginId: loginIdY });
    expect(storage.data.get(AUTH_KEY)).toBe(recordY);
  });

  it("stops a request with SessionChangedError when the renewal after its 401 finds the change first", async () => {
    const { storage, nerve, api, tm } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    // Another tab signs in as Y while the request is out; its storage event reaches this tab only later.
    storage.hold();
    storage.write(AUTH_KEY, recordY);
    await vi.advanceTimersByTimeAsync(0);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");

    // The renewal read Y's record under the lock: it followed Y, refreshed nothing, and stopped the request.
    expect(me.error).toBeInstanceOf(SessionChangedError);
    expect(nerve.calls).toHaveLength(1);
    expect(tm.state).toEqual({ status: "signed-in", loginId: loginIdY });
    expect(storage.data.get(AUTH_KEY)).toBe(recordY);
  });

  it("keeps a request in the session its token was asked for in, though the tab moves on before the token comes", async () => {
    const { storage, nerve, tm } = await setUp();
    // The tab hears another tab's sign-in as Y after the request asked for its token, before the token
    // reaches the request: forced here by an accessToken that writes Y's record as it is called.
    const api = nerve.client();
    api.use(
      authMiddleware(
        {
          get state() {
            return tm.state;
          },
          accessToken: () => {
            const token = tm.accessToken();
            storage.write(AUTH_KEY, recordY);
            return token;
          },
          renew: (sent) => tm.renew(sent),
          endSession: (id) => tm.endSession(id),
        },
        loginId
      )
    );
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBe("Bearer at-1");
    expect(tm.state.loginId).toBe(loginIdY);
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled || nerve.calls.length > 1, "the answer, or another request");

    // The request is X's, whose token it went with: not renewed or sent again in Y's session.
    expect(nerve.calls).toHaveLength(1);
    expect(me.error).toBeInstanceOf(SessionChangedError);
    expect(storage.data.get(AUTH_KEY)).toBe(recordY);
  });

  it("keeps the session the tab moved to when the request sent again is refused, and stops the request", async () => {
    const { storage, nerve, api, tm } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    expect(nerve.calls[2]?.authorization).toBe("Bearer at-2");
    // Another tab signs in as Y while the copy is out; this tab follows.
    storage.write(AUTH_KEY, recordY);
    await until(() => tm.state.loginId === loginIdY, "the switch to Y");
    nerve.calls[2]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");

    // The refused copy was of the first session, whose record Y's sign-in replaced: Y's session stays, and
    // the request stops with SessionChangedError, not as a 401 in Y's session.
    expect(me.error).toBeInstanceOf(SessionChangedError);
    expect(tm.state).toEqual({ status: "signed-in", loginId: loginIdY });
    expect(storage.data.get(AUTH_KEY)).toBe(recordY);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });

  it("stops the request when its copy is refused after another tab signed out, before this tab heard of it", async () => {
    const { storage, nerve, api, tm } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    // Another tab signs out while the copy is out; its storage event reaches this tab only later.
    storage.hold();
    storage.write(AUTH_KEY, null);
    await vi.advanceTimersByTimeAsync(0);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    nerve.calls[2]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");

    // The tab was still in the request's session when the copy was refused; ending it found the record
    // gone, so the sign-out cut the request: it stops with SessionChangedError, not as the copy's 401.
    expect(me.error).toBeInstanceOf(SessionChangedError);
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });

  it.each([
    ["429", () => problem(429, "rate_limited")],
    ["503", () => problem(503, "server_busy")],
    ["no network", () => Response.error()],
  ])("keeps the session when the refresh gets a %s, and fails the request with it", async (_, answer) => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(answer());
    await until(() => me.settled, "the answer");

    expect(me.error).toBeInstanceOf(SessionUnavailableError);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({ refresh_token: "rt-1", login_id: loginId });
    expect(nerve.calls).toHaveLength(2);
  });

  it.each([403, 404, 500])("hands a %s back without renewing", async (status) => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(status, "some_code"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(status);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });

  it("hands a fetch that fails back as it is, without renewing, and keeps the session", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    // No network: a browser's fetch rejects.
    nerve.calls[0]?.fail();
    await until(() => me.settled, "the failure");

    expect(me.error).toBeInstanceOf(TypeError);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({ refresh_token: "rt-1", login_id: loginId });
  });

  it("keeps the session when the fetch of the request sent again fails, and fails the request with it", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    nerve.calls[2]?.fail();
    await until(() => me.settled, "the failure");

    expect(me.error).toBeInstanceOf(TypeError);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({ refresh_token: "rt-2", login_id: loginId });
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });
});
