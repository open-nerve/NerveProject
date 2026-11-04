/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect } from "vitest";
import type { Profile, User } from "@nerve/api-client";
import { json, type Call, type FakeNerve } from "@/lib/auth/fake-nerve";
import { settle } from "@/lib/auth/fake-time";
import type { IUserStore } from "./index";

/** Initialize the real account store through its API; forget only this fixture exchange, not subsequent work. */
export async function seedAccount(nerve: FakeNerve, store: IUserStore, account: User): Promise<void> {
  const previous = nerve.respond;
  let seed: Call | undefined;
  nerve.respond = (call) => {
    if (!seed && call.method === "PATCH" && call.path === "/api/v0/me") {
      seed = call;
      call.answer(json(200, account));
    } else previous?.(call);
  };
  try {
    const answer = await settle(store.updateCurrentUser({ first_name: account.first_name }), "fixture account");
    expect(answer.error).toBeUndefined();
    expect(answer.value).toEqual(account);
  } finally {
    nerve.respond = previous;
    if (seed) nerve.calls.splice(nerve.calls.indexOf(seed), 1);
  }
}

export const profileOf = (): Profile => ({
  theme: "system",
  language: "en",
  start_of_the_week: 0,
  onboarding_step: { profile_complete: false, workspace_create: false, workspace_invite: false, workspace_join: false },
  is_onboarded: false,
  is_tour_completed: false,
  last_workspace_id: null,
  updated_at: "2026-11-04T00:00:00Z",
});
