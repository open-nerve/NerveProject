/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// plane imports
import { useOutsideClickDetector } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
import { EditOutline } from "@makeplane/propel/icons";
import { Loader } from "@plane/ui";
import { cn, renderFormattedDate, getFileURL } from "@plane/utils";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUser } from "@/hooks/store/user";
// local imports
import { useProfileMember } from "./use-profile-member";

type TProfileSidebar = {
  className?: string;
};

export const ProfileSidebar = observer(function ProfileSidebar(props: TProfileSidebar) {
  const { className = "" } = props;
  // refs
  const ref = useRef<HTMLDivElement>(null);
  // router
  const { workspaceSlug, userId } = useParams();
  // store hooks
  const { data: currentUser } = useUser();
  const { profileSidebarCollapsed, toggleProfileSidebar } = useAppTheme();
  const { toggleProfileSettingsModal } = useCommandPalette();
  const profileMember = useProfileMember(workspaceSlug?.toString() ?? "", userId?.toString() ?? "");
  const { t } = useTranslation();

  useOutsideClickDetector(ref, () => {
    if (profileSidebarCollapsed === false) {
      if (window.innerWidth < 768) {
        toggleProfileSidebar();
      }
    }
  });

  useEffect(() => {
    const handleToggleProfileSidebar = () => {
      if (window && window.innerWidth < 768) {
        toggleProfileSidebar(true);
      }
      if (window && profileSidebarCollapsed && window.innerWidth >= 768) {
        toggleProfileSidebar(false);
      }
    };

    window.addEventListener("resize", handleToggleProfileSidebar);
    handleToggleProfileSidebar();
    return () => window.removeEventListener("resize", handleToggleProfileSidebar);
  }, []);

  const renderContent = () => {
    if (profileMember.status === "loading")
      return (
        <Loader className="space-y-7 px-5 py-6">
          <Loader.Item height="52px" width="52px" />
          <div className="space-y-5">
            <Loader.Item height="20px" />
            <Loader.Item height="20px" />
          </div>
        </Loader>
      );
    if (profileMember.status === "load-failed")
      return <div className="px-5 py-6 text-13 text-secondary">{t("profile.details.load_failed")}</div>;
    if (profileMember.status === "not-a-member")
      return <div className="px-5 py-6 text-13 text-secondary">{t("profile.details.not_a_member")}</div>;
    const userData = profileMember.member;
    return (
      <div className="px-5 py-6">
        <div className="flex items-center gap-4">
          <div className="h-[52px] w-[52px] flex-shrink-0 rounded-sm">
            {userData.avatar_url && userData.avatar_url !== "" ? (
              <img
                src={getFileURL(userData.avatar_url)}
                alt={userData.display_name}
                className="h-full w-full rounded-sm object-cover"
              />
            ) : (
              <div className="flex h-[52px] w-[52px] items-center justify-center rounded-sm bg-accent-primary text-on-color capitalize">
                {userData.first_name?.[0]}
              </div>
            )}
          </div>
          <div className="min-w-0">
            <h4 className="truncate text-16 font-semibold">
              {userData.first_name} {userData.last_name}
            </h4>
            <h6 className="truncate text-13 text-secondary">({userData.display_name})</h6>
          </div>
          {currentUser?.id === userId && (
            <div className="ml-auto">
              <IconButton
                variant="secondary"
                icon={EditOutline}
                onClick={() =>
                  toggleProfileSettingsModal({
                    activeTab: "general",
                    isOpen: true,
                  })
                }
              />
            </div>
          )}
        </div>
        <div className="mt-6 flex items-center gap-4 text-13">
          <div className="w-2/5 flex-shrink-0 text-secondary">{t("profile.details.joined_on")}</div>
          <div className="w-3/5 font-medium break-words">{renderFormattedDate(userData.joining_date ?? "")}</div>
        </div>
      </div>
    );
  };

  return (
    <div
      ref={ref}
      className={cn(
        `vertical-scrollbar fixed z-5 scrollbar-md h-full w-full shrink-0 overflow-hidden overflow-y-auto border-l border-subtle bg-surface-1 shadow-raised-200 transition-all md:relative md:w-[300px]`,
        className
      )}
      style={profileSidebarCollapsed ? { marginLeft: `${window?.innerWidth || 0}px` } : {}}
    >
      {renderContent()}
    </div>
  );
});
