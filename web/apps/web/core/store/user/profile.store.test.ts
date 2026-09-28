/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Profile } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { ProfileStore } from "@/store/user/profile.store";

// The profile's changes against a fake nerve that answers each when the test says: a change goes out once the one
// before it is answered or has failed, so nerve applies them in the order they were made and the last answer is
// what nerve holds. The page's language and theme follow the profile (StoreWrapper).

const PROFILE = "/api/v0/me/profile";

/** nerve's answer to a change of the language: the profile with it. */
const profileIn = (language: "en" | "zh-CN") => ({ language, theme: "system" }) as Profile;

/** A store, and two changes made one after the other, to Chinese and then to English: only the first is out. */
async function twoChanges() {
  const nerve = new FakeNerve();
  const store = new ProfileStore({} as RootStore, nerve.client());
  const first = track(store.updateUserProfile({ language: "zh-CN" }));
  const second = track(store.updateUserProfile({ language: "en" }));
  await until(() => nerve.calls.length === 1, "the first change");
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", PROFILE, { language: "zh-CN" }],
  ]);
  return { nerve, store, first, second };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProfileStore.updateUserProfile", () => {
  it("sends a change once nerve has answered the one before it", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(json(200, profileIn("zh-CN")));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.value).toEqual(profileIn("zh-CN"));
    expect(store.data).toEqual(profileIn("zh-CN"));
    expect(nerve.calls[1]?.body).toEqual({ language: "en" });
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(second.value).toEqual(profileIn("en"));
    expect(store.data).toEqual(profileIn("en"));
  });

  it("sends the next change when nerve refuses the one before it, which leaves the profile as it was", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(problem(500, "internal_error"));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(profileIn("en"));
  });

  it("sends the next change when the one before it gets no answer", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.fail();
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(TypeError);
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(profileIn("en"));
  });
});
