/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { OnboardingSteps, Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { afterProfile, resumedPlace, type OnboardingPlace } from "./onboarding-place";

// Where the onboarding resumes and where its profile step leads (M3 design 7.4, decision 2), of the steps nerve keeps
// done and the caller's workspaces.

const none: OnboardingSteps = {
  profile_complete: false,
  workspace_create: false,
  workspace_invite: false,
  workspace_join: false,
};
const named = { ...none, profile_complete: true };
const created = { ...named, workspace_create: true };
// alpha comes first in the list, as nerve orders it by name: zeta is the one the caller created
const alpha = workspaceOf("alpha");
const zeta = workspaceOf("zeta", { role: 20 });
const { PROFILE_SETUP, WORKSPACE_CREATE_OR_JOIN, INVITE_MEMBERS } = EOnboardingSteps;

describe("resumedPlace", () => {
  it.each<{ account: string; steps: OnboardingSteps; workspaces: Workspace[]; place: OnboardingPlace }>([
    { account: "a new account", steps: none, workspaces: [], place: { kind: PROFILE_SETUP } },
    {
      account: "one named, with no workspace",
      steps: named,
      workspaces: [],
      place: { kind: WORKSPACE_CREATE_OR_JOIN },
    },
    { account: "one named, invited since", steps: named, workspaces: [alpha], place: { kind: PROFILE_SETUP } },
    {
      account: "one that created a workspace and invited no one",
      steps: created,
      workspaces: [alpha, zeta],
      place: { kind: INVITE_MEMBERS, workspace: zeta },
    },
    {
      account: "one that created a workspace and invited",
      steps: { ...created, workspace_invite: true },
      workspaces: [alpha, zeta],
      place: { kind: PROFILE_SETUP },
    },
    {
      account: "one whose workspace created is gone",
      steps: created,
      workspaces: [alpha],
      place: { kind: PROFILE_SETUP },
    },
  ])("resumes $account at $place.kind", ({ steps, workspaces, place }) => {
    expect(resumedPlace({ onboarding_step: steps, last_workspace_id: zeta.id }, workspaces)).toEqual(place);
  });
});

describe("afterProfile", () => {
  it.each<{ account: string; workspaces: Workspace[]; next: "finish" | "create" }>([
    { account: "one with no workspace", workspaces: [], next: "create" },
    { account: "one with a workspace", workspaces: [alpha], next: "finish" },
  ])("leads $account to $next", ({ workspaces, next }) => {
    expect(afterProfile(workspaces)).toBe(next);
  });
});
