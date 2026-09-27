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

// The profile's updates against a fake nerve that answers them in the order each test sets: the profile holds the
// answer to the newest update nerve accepted, whichever answer lands last. The page's language and theme follow
// the profile (StoreWrapper), so an older answer landing last would take them back.

const PROFILE = "/api/v0/me/profile";

/** nerve's answer to a change of the language: the profile with it. */
const profileIn = (language: "en" | "zh-CN") => ({ language, theme: "system" }) as Profile;

/** A store, and the two updates it sends, to English and then to Chinese, both out. */
async function twoUpdatesOut() {
  const nerve = new FakeNerve();
  const store = new ProfileStore({} as RootStore, nerve.client());
  const older = track(store.updateUserProfile({ language: "en" }));
  const newer = track(store.updateUserProfile({ language: "zh-CN" }));
  await until(() => nerve.calls.length === 2, "both updates");
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", PROFILE, { language: "en" }],
    ["PATCH", PROFILE, { language: "zh-CN" }],
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

describe("ProfileStore.updateUserProfile", () => {
  it("keeps the newer answer when nerve answers the newer update first", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerNewer(json(200, profileIn("zh-CN")));
    await until(() => newer.settled, "the newer answer");
    expect(store.data).toEqual(profileIn("zh-CN"));
    answerOlder(json(200, profileIn("en")));
    await until(() => older.settled, "the older answer");

    // The older answer lands last, and is dropped; each caller still gets its own answer.
    expect(store.data).toEqual(profileIn("zh-CN"));
    expect(older.value).toEqual(profileIn("en"));
    expect(newer.value).toEqual(profileIn("zh-CN"));
  });

  it("keeps the older answer when nerve refuses the newer update and accepts the older one", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerNewer(problem(500, "internal_error"));
    await until(() => newer.settled, "the refusal");
    expect(newer.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    answerOlder(json(200, profileIn("en")));
    await until(() => older.settled, "the older answer");

    expect(store.data).toEqual(profileIn("en"));
  });

  it("keeps the newer answer when nerve answers in order", async () => {
    const { store, older, newer, answerOlder, answerNewer } = await twoUpdatesOut();

    answerOlder(json(200, profileIn("en")));
    await until(() => older.settled, "the older answer");
    expect(store.data).toEqual(profileIn("en"));
    answerNewer(json(200, profileIn("zh-CN")));
    await until(() => newer.settled, "the newer answer");

    expect(store.data).toEqual(profileIn("zh-CN"));
  });
});
