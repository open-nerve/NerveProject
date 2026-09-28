/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { UserStore } from "@/store/user";

// The account's changes (names, time zone) against a fake nerve that answers each when the test says: a change goes
// out once the one before it is answered or has failed, so nerve applies them in the order they were made and the
// last answer is what nerve holds. The store gets its session's client from RootStore; the token manager it
// imports is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const ME = "/api/v0/me";

/** nerve's answer to a change of the time zone: the account in it. */
const accountIn = (user_timezone: string) => ({ first_name: "Ada", user_timezone }) as User;

/** A store, and two changes made one after the other, to Shanghai and then to Berlin: only the first is out. */
async function twoChanges() {
  const nerve = new FakeNerve();
  const store = new UserStore({} as RootStore, nerve.client());
  const first = track(store.updateCurrentUser({ user_timezone: "Asia/Shanghai" }));
  const second = track(store.updateCurrentUser({ user_timezone: "Europe/Berlin" }));
  await until(() => nerve.calls.length === 1, "the first change");
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", ME, { user_timezone: "Asia/Shanghai" }],
  ]);
  return { nerve, store, first, second };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserStore.updateCurrentUser", () => {
  it("sends a change once nerve has answered the one before it", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(json(200, accountIn("Asia/Shanghai")));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.value).toEqual(accountIn("Asia/Shanghai"));
    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
    expect(nerve.calls[1]?.body).toEqual({ user_timezone: "Europe/Berlin" });
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(second.value).toEqual(accountIn("Europe/Berlin"));
    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when nerve refuses the one before it, which leaves the account as it was", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(problem(500, "internal_error"));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when the one before it gets no answer", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.fail();
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(TypeError);
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });
});
