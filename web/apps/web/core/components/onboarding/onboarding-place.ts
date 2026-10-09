/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Profile, Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";

/** Where the onboarding is (M3 design 7.4): a step; the invitation step's with the workspace it invites to. */
export type OnboardingPlace =
  | { kind: EOnboardingSteps.PROFILE_SETUP }
  | { kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN }
  | { kind: EOnboardingSteps.INVITE_MEMBERS; workspace: Workspace };

/**
 * Where the onboarding of profile resumes, by the steps nerve keeps done (M3 design 7.4): at the invitation step, for
 * the workspace it created (the one it opened last, written with the step), while the caller has it and has invited
 * no one yet; at the creation's, for one who has no workspace and created none; else at the profile step, which is
 * the last for one who has a workspace (afterProfile).
 */
export function resumedPlace(
  profile: Pick<Profile, "onboarding_step" | "last_workspace_id">,
  workspaces: readonly Workspace[]
): OnboardingPlace {
  const steps = profile.onboarding_step;
  const created = workspaces.find((workspace) => workspace.id === profile.last_workspace_id);
  if (steps.profile_complete && steps.workspace_create && !steps.workspace_invite && created)
    return { kind: EOnboardingSteps.INVITE_MEMBERS, workspace: created };
  if (steps.profile_complete && !steps.workspace_create && workspaces.length === 0)
    return { kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN };
  return { kind: EOnboardingSteps.PROFILE_SETUP };
}

/**
 * What follows the profile step (M3 design 7.4, decision 2): the end, for one who has a workspace (the invited, who
 * accepted first); else the creation of one.
 */
export const afterProfile = (workspaces: readonly Workspace[]): "finish" | "create" =>
  workspaces.length > 0 ? "finish" : "create";
