/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// plane imports
import { ContentWrapper } from "@nerve/ui";
// hooks
import { useUserProfile, useUser } from "@/hooks/store/user";
// plane web imports
import { TourRoot } from "@/components/onboarding/tour/root";
// local imports
import { HomeBody } from "./home-body";
import { UserGreetingsView } from "./user-greetings";
import { HomePeekOverviewsRoot } from "../issues/peek-overview/peek-overviews";

export const WorkspaceHomeView = observer(function WorkspaceHomeView() {
  // store hooks
  const { data: currentUser } = useUser();
  const { data: currentUserProfile, updateTourCompleted } = useUserProfile();

  const handleTourCompleted = async () => {
    try {
      await updateTourCompleted();
    } catch (error) {
      console.error("Error updating tour completed", error);
    }
  };

  // TODO: refactor loader implementation
  return (
    <>
      {currentUserProfile && !currentUserProfile.is_tour_completed && (
        <div className="fixed top-0 left-0 z-20 grid h-full w-full place-items-center overflow-y-auto bg-backdrop transition-opacity">
          <TourRoot onComplete={handleTourCompleted} />
        </div>
      )}
      <>
        <HomePeekOverviewsRoot />
        <ContentWrapper className="mx-auto scrollbar-hide gap-6 bg-surface-1 px-page-x">
          <div className="mx-auto w-full max-w-[800px]">
            {currentUser && <UserGreetingsView user={currentUser} />}
            <HomeBody />
          </div>
        </ContentWrapper>
      </>
    </>
  );
});
