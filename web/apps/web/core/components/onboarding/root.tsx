/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { OnboardingStepsUpdate } from "@nerve/api-client";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IWorkspaceMemberInvitation, TOnboardingStep } from "@nerve/types";
import { EOnboardingSteps } from "@nerve/types";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserProfile } from "@/hooks/store/user";
// local components
import { OnboardingHeader } from "./header";
import { OnboardingStepRoot } from "./steps";

type Props = {
  invitations?: IWorkspaceMemberInvitation[];
};

export const OnboardingRoot = observer(function OnboardingRoot({ invitations = [] }: Props) {
  const [currentStep, setCurrentStep] = useState<TOnboardingStep>(EOnboardingSteps.PROFILE_SETUP);
  // store hooks
  const { data: user } = useUser();
  const { data: userProfile, updateUserProfile, finishUserOnboarding } = useUserProfile();
  const { workspaces } = useWorkspace();

  const workspacesList = Object.values(workspaces ?? {});

  // Calculate total steps based on whether invitations are available
  const hasInvitations = invitations.length > 0;

  // complete onboarding
  const finishOnboarding = useCallback(async () => {
    if (!user) return;
    try {
      await finishUserOnboarding();
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Failed",
        message: "Failed to finish onboarding, Please try again later.",
      });
    }
  }, [user, finishUserOnboarding]);

  // handle step change: nerve merges the steps it is given into the profile's
  const stepChange = useCallback(
    async (steps: OnboardingStepsUpdate) => {
      if (!user) return;
      try {
        await updateUserProfile({ onboarding_step: steps });
      } catch {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Failed",
          message: "Failed to save your progress, Please try again later.",
        });
      }
    },
    [user, updateUserProfile]
  );

  // finishing sets all four steps in its one write, so it goes without the step change it supersedes: two
  // profile writes in flight would leave the store with whichever answer lands last
  const handleStepChange = useCallback(
    (step: EOnboardingSteps, skipInvites?: boolean) => {
      switch (step) {
        case EOnboardingSteps.PROFILE_SETUP:
          if (workspacesList.length > 0) finishOnboarding();
          else {
            stepChange({ profile_complete: true });
            setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
          }
          break;
        case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
          if (skipInvites) finishOnboarding();
          else {
            setCurrentStep(EOnboardingSteps.INVITE_MEMBERS);
            stepChange({ workspace_create: true });
          }
          break;
        case EOnboardingSteps.INVITE_MEMBERS:
          finishOnboarding();
          break;
      }
    },
    [stepChange, finishOnboarding, workspacesList]
  );

  const updateCurrentStep = (step: EOnboardingSteps) => setCurrentStep(step);

  useEffect(() => {
    const handleInitialStep = () => {
      if (
        userProfile?.onboarding_step?.profile_complete &&
        !userProfile?.onboarding_step?.workspace_create &&
        !userProfile?.onboarding_step?.workspace_join
      ) {
        setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
      }
      if (
        userProfile?.onboarding_step?.profile_complete &&
        userProfile?.onboarding_step?.workspace_create &&
        !userProfile?.onboarding_step?.workspace_invite
      ) {
        setCurrentStep(EOnboardingSteps.INVITE_MEMBERS);
      }
    };

    handleInitialStep();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex h-full flex-col">
      {/* Header with progress */}
      <OnboardingHeader
        currentStep={currentStep}
        updateCurrentStep={updateCurrentStep}
        hasInvitations={hasInvitations}
      />

      {/* Main content area */}
      <OnboardingStepRoot currentStep={currentStep} invitations={invitations} handleStepChange={handleStepChange} />
    </div>
  );
});
