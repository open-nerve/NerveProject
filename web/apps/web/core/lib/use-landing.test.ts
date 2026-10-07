/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OnboardingSteps, Profile, Workspace } from "@nerve/api-client";
import { EPageTypes } from "@/helpers/authentication.helper";
import { handed, response } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The landing of a signed-in account (M3 design 3.14) as AuthenticationWrapper is told it, with fake-session-swr.ts's
// stand-in for useSessionSWR and stand-ins for the stores. Where the landing goes, of the caller's workspaces, is
// landing.test.ts.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const stores = vi.hoisted(
  (): { profile: Profile | undefined; workspaces: Workspace[] | undefined; calls: string[] } => ({
    profile: undefined,
    workspaces: undefined,
    calls: [],
  })
);
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: stores.profile }) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    workspaces: stores.workspaces,
    fetchWorkspaces: () => Promise.resolve(stores.calls.push("the workspaces")),
  }),
}));

const { useLanding } = await import("./use-landing");

const done: OnboardingSteps = {
  profile_complete: true,
  workspace_create: true,
  workspace_invite: true,
  workspace_join: true,
};

/** The caller's profile as nerve gives it: onboarded, and the workspace he opened last is beta, unless fields say not. */
function profileOf(fields: Partial<Profile> = {}): Profile {
  return {
    theme: "system",
    language: "en",
    start_of_the_week: 0,
    onboarding_step: done,
    is_onboarded: true,
    is_tour_completed: false,
    last_workspace_id: "id-beta",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}
const acme = workspaceOf("acme", { created_at: "2026-09-01T09:00:00Z" });
const beta = workspaceOf("beta", { created_at: "2026-09-02T09:00:00Z" });
const { AUTHENTICATED, NON_AUTHENTICATED, ONBOARDING, PUBLIC } = EPageTypes;

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  stores.profile = profileOf();
  stores.workspaces = [acme, beta];
  stores.calls = [];
});

describe("useLanding", () => {
  it("sends an onboarded account on the sign-in page where its workspaces decide, the one it opened last first", async () => {
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "go", to: "/beta" });
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher()));
    expect(stores.calls).toEqual(["the workspaces"]);

    stores.profile = profileOf({ last_workspace_id: null });
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "go", to: "/acme" });
  });

  it("lands an onboarded account on the onboarding page the same way", () => {
    expect(useLanding(ONBOARDING, undefined)).toEqual({ kind: "go", to: "/beta" });
  });

  it.each([
    {
      who: "an account whose profile has not come",
      pageType: NON_AUTHENTICATED,
      nextPath: undefined,
      profile: undefined,
    },
    {
      who: "an account still onboarding",
      pageType: NON_AUTHENTICATED,
      nextPath: undefined,
      profile: profileOf({ is_onboarded: false, onboarding_step: { ...done, workspace_create: false } }),
    },
    {
      who: "an account with a valid next_path",
      pageType: NON_AUTHENTICATED,
      nextPath: "/acme/projects",
      profile: profileOf(),
    },
    { who: "an account on a page that needs one", pageType: AUTHENTICATED, nextPath: undefined, profile: profileOf() },
    { who: "an account on a public page", pageType: PUBLIC, nextPath: undefined, profile: profileOf() },
  ])("leaves $who to the wrapper's other rules, and fetches nothing", ({ pageType, nextPath, profile }) => {
    stores.profile = profile;
    expect(useLanding(pageType, nextPath)).toEqual({ kind: "none" });
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
  });

  it("waits for the caller's workspaces, and says when nerve cannot give them, with a retry", () => {
    stores.workspaces = undefined;
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "loading" });

    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: new Error("nerve cannot be reached"), mutate };
    const landing = useLanding(NON_AUTHENTICATED, undefined);
    expect(landing.kind).toBe("unavailable");
    if (landing.kind === "unavailable") landing.retry();
    expect(mutate).toHaveBeenCalledOnce();
  });
});
