/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
// nerve imports
import type { Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";
// local components
import type { OnboardingPlace } from "../onboarding-place";
import { ProfileSetupStep } from "./profile";
import { InviteTeamStep } from "./team";
import { WorkspaceCreateStep } from "./workspace";

type Props = {
  place: OnboardingPlace;
  /**
   * The profile step is done. When that ends the onboarding, the step stays busy until what this returns settles, so
   * that no second click sends the names and the end again.
   */
  onNamed: () => Promise<void> | undefined;
  /**
   * The creation step created workspace; alone when it is for its creator alone. The step stays busy until this
   * settles, so that no second click checks the new workspace's slug again.
   */
  onCreated: (workspace: Workspace, alone: boolean) => Promise<void>;
  /** The invitation step is done, or put off: the step stays busy until this settles, as the onboarding ends. */
  onDone: () => Promise<void>;
};

function OnboardingStepContent({ place, onNamed, onCreated, onDone }: Props) {
  switch (place.kind) {
    case EOnboardingSteps.PROFILE_SETUP:
      return <ProfileSetupStep onDone={onNamed} />;
    case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
      return <WorkspaceCreateStep onCreated={onCreated} />;
    case EOnboardingSteps.INVITE_MEMBERS:
      return <InviteTeamStep workspace={place.workspace} onDone={onDone} />;
  }
}

export function OnboardingStepRoot(props: Props) {
  const { place } = props;
  // ref for the scrollable container
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // scroll to top when step changes
  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({
        top: 0,
        behavior: "smooth",
      });
    }
  }, [place.kind]);

  return (
    <div ref={scrollContainerRef} className="flex-1 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-8">
        <div className="w-full max-w-[24rem]">
          <OnboardingStepContent {...props} />
        </div>
      </div>
    </div>
  );
}
