/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet } from "react-router";
// components
import { PROFILE_TABS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProfileSidebar } from "@/components/profile/sidebar";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
import useSize from "@/hooks/use-window-size";
import { parseProfileView } from "@/lib/profile-view";
// local components
import type { Route } from "./+types/layout";
import { UserProfileHeader } from "./header";
import { ProfileIssuesMobileHeader } from "./mobile-header";
import { ProfileNavbar } from "./navbar";

function UseProfileLayout({ params }: Pick<Route.ComponentProps, "params">) {
  // router
  const { workspaceSlug, userId, profileViewId } = params;
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();
  // derived values
  const isAuthorized = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
  );

  const windowSize = useSize();
  const isSmallerScreen = windowSize[0] >= 768;

  // derived values
  const currentTab = PROFILE_TABS.find((tab) => tab.route === profileViewId);
  const showIssues = !!isAuthorized && parseProfileView(profileViewId) !== undefined;

  return (
    <Fragment key={JSON.stringify([workspaceSlug, userId, profileViewId])}>
      {/* Passing the current tab from the route value as we need the header as top most component. */}
      <div className="flex h-full w-full flex-col overflow-hidden md:flex-row">
        <div className="flex h-full w-full flex-col overflow-hidden">
          <AppHeader
            header={<UserProfileHeader currentTab={currentTab} showProfileIssuesFilter={showIssues} />}
            mobileHeader={showIssues && <ProfileIssuesMobileHeader />}
          />
          <ContentWrapper>
            <div className="flex h-full w-full flex-row md:flex-col md:overflow-hidden">
              <div className="flex w-full flex-col md:h-full md:overflow-hidden">
                <ProfileNavbar isAuthorized={!!isAuthorized} />
                {isAuthorized ? (
                  <div className="h-full w-full overflow-hidden">
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
    </Fragment>
  );
}

export default observer(UseProfileLayout);
