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

// The account's updates (names, time zone) against a fake nerve that answers them in the order each test sets: the
// account holds the answer to the newest update nerve accepted, whichever answer lands last. The store gets its
// session's client from RootStore; the token manager it imports is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const ME = "/api/v0/me";

/** nerve's answer to a change of the time zone: the account in it. */
const accountIn = (user_timezone: string) => ({ first_name: "Ada", user_timezone }) as User;

/** A store, and the two updates it sends, to Berlin and then to Shanghai, both out. */
async function twoUpdatesOut() {
  const nerve = new FakeNerve();
  const store = new UserStore({} as RootStore, nerve.client());
  const older = track(store.updateCurrentUser({ user_timezone: "Europe/Berlin" }));
  const newer = track(store.updateCurrentUser({ user_timezone: "Asia/Shanghai" }));
  await until(() => nerve.calls.length === 2, "both updates");
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", ME, { user_timezone: "Europe/Berlin" }],
    ["PATCH", ME, { user_timezone: "Asia/Shanghai" }],
  ]);
  const [answerOlder, answerNewer] = nerve.calls.map((call) => call.answer);
  return { store, older, newer, answerOlder: answerOlder!, answerNewer: answerNewer! };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserStore.updateCurrentUser", () => {
  it("keeps the newer answer when nerve answers the newer update first", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerNewer(json(200, accountIn("Asia/Shanghai")));
    await until(() => newer.settled, "the newer answer");
    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
    answerOlder(json(200, accountIn("Europe/Berlin")));
    await until(() => older.settled, "the older answer");

    // The older answer lands last, and is dropped; each caller still gets its own answer.
    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
    expect(older.value).toEqual(accountIn("Europe/Berlin"));
    expect(newer.value).toEqual(accountIn("Asia/Shanghai"));
  });

  it("keeps the older answer when nerve refuses the newer update and accepts the older one", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerNewer(problem(500, "internal_error"));
    await until(() => newer.settled, "the refusal");
    expect(newer.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    answerOlder(json(200, accountIn("Europe/Berlin")));
    await until(() => older.settled, "the older answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("keeps the newer answer when nerve answers in order", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerOlder(json(200, accountIn("Europe/Berlin")));
    await until(() => older.settled, "the older answer");
    expect(store.data).toEqual(accountIn("Europe/Berlin"));
    answerNewer(json(200, accountIn("Asia/Shanghai")));
    await until(() => newer.settled, "the newer answer");

    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
  });
});
