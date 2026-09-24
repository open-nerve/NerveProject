/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import { observer } from "mobx-react";
import { useParams, useNavigate } from "react-router";
import { ChevronDownOutline, RightSidePaneOutline, YourWorkOutline } from "@makeplane/propel/icons";
import { PROFILE_TABS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Breadcrumbs, Header, CustomMenu } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ProfileIssuesFilter } from "@/components/profile/profile-issues-filter";
import { useProfileMember } from "@/components/profile/use-profile-member";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { Button } from "@nerve/propel/button";

type TUserProfileHeader = {
  type?: string | undefined;
  showProfileIssuesFilter?: boolean;
};

export const UserProfileHeader = observer(function UserProfileHeader(props: TUserProfileHeader) {
  const { type = undefined, showProfileIssuesFilter } = props;
  // router
  const { workspaceSlug, userId } = useParams();
  const navigate = useNavigate();
  // store hooks
  const { toggleProfileSidebar, profileSidebarCollapsed } = useAppTheme();
  const { data: currentUser } = useUser();
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
  const { member } = useProfileMember(workspaceSlug ?? "", userId ?? "");
  const { t } = useTranslation();
  // derived values
  const isAuthorized = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );

  if (!workspaceUserInfo) return null;

  const tabsList = isAuthorized ? PROFILE_TABS : [];

  const userName = member ? `${member.first_name} ${member.last_name}`.trim() : "";

  const isCurrentUser = currentUser?.id === userId;

  // There is no name to show until the members have loaded, or for a user who is not (or no longer) a member.
  const breadcrumbLabel = isCurrentUser
    ? t("profile.page_label")
    : userName
      ? `${userName} ${t("profile.work")}`
      : t("profile.work");

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs>
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={breadcrumbLabel}
                disableTooltip
                icon={<YourWorkOutline className="h-4 w-4 text-tertiary" />}
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <div className="hidden md:flex md:items-center">{showProfileIssuesFilter && <ProfileIssuesFilter />}</div>
        <div className="flex gap-4 md:hidden">
          <CustomMenu
            maxHeight={"md"}
            className="flex flex-grow justify-center text-13 text-secondary"
            placement="bottom-start"
            customButton={
              <div className="flex items-center gap-2 rounded-md border border-subtle px-2 py-1.5">
                <span className="flex flex-grow justify-center text-13 text-secondary">{type}</span>
                <ChevronDownOutline className="h-4 w-4 text-placeholder" />
              </div>
            }
            customButtonClassName="flex flex-grow justify-center text-secondary text-13"
            closeOnSelect
          >
            <></>
            {tabsList.map((tab) => (
              <CustomMenu.MenuItem
                className="flex items-center gap-2"
                key={tab.route}
                onClick={() => navigate(`/${workspaceSlug}/profile/${userId}/${tab.route}`)}
              >
                <span className="w-full text-tertiary">{t(tab.i18n_label)}</span>
              </CustomMenu.MenuItem>
            ))}
          </CustomMenu>
          {/* On a small screen the user card closes on a click outside it. This button toggles the card from
              outside it, so it opts out: otherwise its mousedown would close the card and its click reopen it. */}
          <div className="shrink-0 md:hidden" data-prevent-outside-click>
            <Button
              variant="ghost"
              size="lg"
              onClick={() => {
                toggleProfileSidebar();
              }}
              appendIcon={
                <RightSidePaneOutline className={!profileSidebarCollapsed ? "text-accent-primary" : "text-secondary"} />
              }
            ></Button>
          </div>
        </div>
      </Header.RightItem>
    </Header>
  );
});
