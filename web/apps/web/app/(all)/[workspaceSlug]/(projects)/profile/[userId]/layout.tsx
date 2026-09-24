/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Outlet } from "react-router";
// components
import { PROFILE_TABS, EUserPermissions, EUserPermissionsLevel } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProfileSidebar } from "@/components/profile/sidebar";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
import useSize from "@/hooks/use-window-size";
// local components
import type { Route } from "./+types/layout";
import { UserProfileHeader } from "./header";
import { ProfileIssuesMobileHeader } from "./mobile-header";
import { ProfileNavbar } from "./navbar";

function UseProfileLayout({ params }: Route.ComponentProps) {
  // router
  const { profileViewId } = params;
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();
  // derived values
  const isAuthorized = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );

  const windowSize = useSize();
  const isSmallerScreen = windowSize[0] >= 768;

  // derived values
  const currentTab = PROFILE_TABS.find((tab) => tab.route === profileViewId);
  const isIssuesTab = currentTab !== undefined;

  return (
    <>
      {/* Passing the type prop from the current route value as we need the header as top most component. */}
      <div className="flex h-full w-full flex-col overflow-hidden md:flex-row">
        <div className="flex h-full w-full flex-col overflow-hidden">
          <AppHeader
            header={<UserProfileHeader type={currentTab?.i18n_label} showProfileIssuesFilter={isIssuesTab} />}
            mobileHeader={isIssuesTab && <ProfileIssuesMobileHeader />}
          />
          <ContentWrapper>
            <div className="flex h-full w-full flex-row md:flex-col md:overflow-hidden">
              <div className="flex w-full flex-col md:h-full md:overflow-hidden">
                <ProfileNavbar isAuthorized={!!isAuthorized} />
                {isAuthorized ? (
                  <div className={`h-full w-full overflow-hidden`}>
                    <Outlet />
                  </div>
                ) : (
                  <div className="grid h-full w-full place-items-center text-secondary">
                    {t("you_do_not_have_the_permission_to_access_this_page")}
                  </div>
                )}
              </div>
              {!isSmallerScreen && <ProfileSidebar />}
            </div>
          </ContentWrapper>
        </div>
        {isSmallerScreen && <ProfileSidebar />}
      </div>
    </>
  );
}

export default observer(UseProfileLayout);
