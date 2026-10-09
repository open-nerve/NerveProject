/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { ProfileUpdate, Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { LogoSpinner } from "@/components/common/logo-spinner";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserProfile } from "@/hooks/store/user";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local components
import { OnboardingHeader } from "./header";
import { afterProfile, resumedPlace, type OnboardingPlace } from "./onboarding-place";
import { OnboardingStepRoot } from "./steps";

type Props = {
  /** The caller's workspaces, as nerve listed them. */
  workspaces: Workspace[];
};

const OnboardingSteps = observer(function OnboardingSteps({ workspaces }: Props) {
  // store hooks
  const { data: profile, updateUserProfile, finishUserOnboarding } = useUserProfile();
  // where the onboarding is: where it was left, as it opens
  const [place, setPlace] = useState<OnboardingPlace>(() =>
    profile ? resumedPlace(profile, workspaces) : { kind: EOnboardingSteps.PROFILE_SETUP }
  );

  // a change of the profile; nerve's refusal says why, in the session it was sent in alone (M3 design 7.1)
  const failed = useRefusalToast();
  const change = (data: ProfileUpdate) => void followInSession(() => updateUserProfile(data), { failed });
  // settles once nerve has answered, and never rejects (followInSession); a refusal is onFailed's, by default failed's
  const finish = (onFailed: (error: unknown) => void = failed) =>
    followInSession(() => finishUserOnboarding(), { failed: onFailed });

  // one who has a workspace is done after the profile step, which stays busy until nerve has answered the end; one who
  // has none creates one (M3 design 7.4)
  const named = () => {
    if (afterProfile(workspaces) === "finish") return finish();
    change({ onboarding_step: { profile_complete: true } });
    setPlace({ kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN });
  };

  // the workspace created is the one opened last, and the one the invitation step invites to. A workspace for its
  // creator alone has no invitation step: the onboarding ends, and the creation step stays busy until nerve has
  // answered the end, which goes after the step's change (the profile's changes are sent one at a time). An end nerve
  // refuses says why, and moves on to the workspace's invitation step all the same, as a reload would: the creation
  // is over once the workspace exists, and that step's "later" ends the onboarding again.
  const created = async (workspace: Workspace, alone: boolean) => {
    change({ onboarding_step: { workspace_create: true }, last_workspace_id: workspace.id });
    const invite = () => setPlace({ kind: EOnboardingSteps.INVITE_MEMBERS, workspace });
    if (!alone) {
      invite();
      return;
    }
    await finish((error) => {
      failed(error);
      invite();
    });
  };

  return (
    <div className="flex h-full flex-col">
      {/* Header with progress: its one way back is from the creation to the profile step */}
      <OnboardingHeader
        currentStep={place.kind}
        updateCurrentStep={() => setPlace({ kind: EOnboardingSteps.PROFILE_SETUP })}
      />

      {/* Main content area */}
      <OnboardingStepRoot place={place} onNamed={named} onCreated={created} onDone={() => finish()} />
    </div>
  );
});

/**
 * The onboarding (M3 design 7.4): its steps once nerve has listed the caller's workspaces, which decide them. The list
 * is the session's, the one the landing reads once the onboarding is done.
 */
export const OnboardingRoot = observer(function OnboardingRoot() {
  // store hooks
  const { workspaces } = useWorkspace();
  const listed = useWorkspacesFetch();

  if (workspaces) return <OnboardingSteps workspaces={workspaces} />;
  if (listed.error) return <SessionUnavailable autoRetry={false} onRetry={() => void listed.mutate()} />;
  return (
    <div className="grid h-full w-full place-items-center">
      <LogoSpinner />
    </div>
  );
});
