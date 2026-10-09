/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Profile } from "@nerve/api-client";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserProfile } from "@/hooks/store/user";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
// lib
import { landingPath } from "@/lib/landing";

/** Whether the account is done with onboarding (M2 design 7.4): its profile says so, or every step is done. */
export const isOnboarded = (profile: Profile) =>
  profile.is_onboarded ||
  (profile.onboarding_step.profile_complete &&
    profile.onboarding_step.workspace_create &&
    profile.onboarding_step.workspace_invite &&
    profile.onboarding_step.workspace_join);

/**
 * What AuthenticationWrapper does about the landing: nothing, where the page does not land the account; wait for the
 * caller's workspaces; say nerve cannot be reached, with a retry; or go to the landing.
 */
export type Landing =
  | { kind: "none" }
  | { kind: "loading" }
  | { kind: "unavailable"; retry: () => void }
  | { kind: "go"; to: string };

/**
 * The landing of a signed-in account (M3 design 3.14), the whole decision: an onboarded account that the sign-in or
 * sign-up page, or the onboarding page, sends on without a valid next_path goes where its workspaces decide, the one
 * it opened last first (landingPath). The caller's workspaces are fetched, for the session, only then.
 */
export function useLanding(pageType: EPageTypes, validNextPath: string | undefined): Landing {
  const { data: profile } = useUserProfile();
  const { workspaces } = useWorkspace();
  const lands =
    profile !== undefined &&
    isOnboarded(profile) &&
    !validNextPath &&
    (pageType === EPageTypes.NON_AUTHENTICATED || pageType === EPageTypes.ONBOARDING);
  const listed = useWorkspacesFetch(lands);
  if (!lands) return { kind: "none" };
  if (listed.error) return { kind: "unavailable", retry: () => void listed.mutate() };
  if (!workspaces) return { kind: "loading" };
  return { kind: "go", to: landingPath(workspaces, profile.last_workspace_id) };
}
