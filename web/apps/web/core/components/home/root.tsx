/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { ContentWrapper } from "@nerve/ui";
// hooks
import { useUserProfile, useUser } from "@/hooks/store/user";
// components
import { TourRoot } from "@/components/onboarding/tour/root";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
// local imports
import { HomeBody } from "./home-body";
import { UserGreetingsView } from "./user-greetings";

export const WorkspaceHomeView = observer(function WorkspaceHomeView() {
  // store hooks
  const { data: currentUser } = useUser();
  const { data: currentUserProfile, updateTourCompleted } = useUserProfile();
  const { t } = useTranslation();

  // the tour ends once nerve has its end in the profile; a refusal's reason is said, and the tour stays. Followed only
  // in the session the end was sent in (M3 design 7.1)
  const completeTour = () =>
    void followInSession(() => updateTourCompleted(), {
      failed: (error) =>
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) }),
    });

  // TODO: refactor loader implementation
  return (
    <>
      {currentUserProfile && !currentUserProfile.is_tour_completed && (
        <div className="fixed top-0 left-0 z-20 grid h-full w-full place-items-center overflow-y-auto bg-backdrop transition-opacity">
          <TourRoot onComplete={completeTour} />
        </div>
      )}
      <ContentWrapper className="mx-auto scrollbar-hide gap-6 bg-surface-1 px-page-x">
        <div className="mx-auto w-full max-w-[800px]">
          {currentUser && <UserGreetingsView user={currentUser} />}
          <HomeBody />
        </div>
      </ContentWrapper>
    </>
  );
});
