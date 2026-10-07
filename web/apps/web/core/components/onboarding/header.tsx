/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { ChevronLeftOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { TOnboardingStep } from "@nerve/types";
import { EOnboardingSteps } from "@nerve/types";
import { cn } from "@nerve/utils";
// components
import { NerveLockup } from "@/components/common/nerve-logo";
// hooks
import { useUser } from "@/hooks/store/user";
// local imports
import { SwitchAccountDropdown } from "./switch-account-dropdown";

type OnboardingHeaderProps = {
  currentStep: EOnboardingSteps;
  updateCurrentStep: (step: EOnboardingSteps) => void;
};

export const OnboardingHeader = observer(function OnboardingHeader(props: OnboardingHeaderProps) {
  const { currentStep, updateCurrentStep } = props;
  // store hooks
  const { data: user } = useUser();

  // handle step back
  const handleStepBack = () => {
    if (currentStep === EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN) updateCurrentStep(EOnboardingSteps.PROFILE_SETUP);
  };

  // can go back
  const canGoBack = ![EOnboardingSteps.PROFILE_SETUP, EOnboardingSteps.INVITE_MEMBERS].includes(currentStep);

  // step order for progress tracking
  const stepOrder: TOnboardingStep[] = [
    EOnboardingSteps.PROFILE_SETUP,
    EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN,
    EOnboardingSteps.INVITE_MEMBERS,
  ];

  // derived values
  const currentStepNumber = stepOrder.indexOf(currentStep) + 1;
  const totalSteps = stepOrder.length;
  const userName = user?.display_name
    ? user?.display_name
    : user?.first_name
      ? `${user?.first_name} ${user?.last_name ?? ""}`
      : user?.email;

  return (
    <div className="sticky top-0 z-10 flex flex-col gap-4">
      <div className="h-1.5 w-full cursor-pointer overflow-hidden rounded-t-lg bg-surface-1">
        <Tooltip label={`${currentStepNumber}/${totalSteps}`} side="bottom" align="end">
          <div
            className="h-full bg-accent-primary transition-all duration-700 ease-out"
            style={{ width: `${(currentStepNumber / totalSteps) * 100}%` }}
          />
        </Tooltip>
      </div>
      <div className={cn("flex w-full items-center justify-between gap-6 px-6", canGoBack && "pr-6 pl-4")}>
        <div className="flex items-center gap-2.5">
          {canGoBack && (
            <button onClick={handleStepBack} className="cursor-pointer" type="button" disabled={!canGoBack}>
              <ChevronLeftOutline className="size-6 text-placeholder" />
            </button>
          )}
          <NerveLockup className="h-5 w-auto" />
        </div>
        <SwitchAccountDropdown fullName={userName} />
      </div>
    </div>
  );
});
