/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { Ban } from "lucide-react";
import { ProfileAvatar } from "@/components/assets/profile-image";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
import { useTranslation } from "@nerve/i18n";
import { EUserProjectRoles } from "@nerve/types";
// nerve ui
import { CustomSearchSelect } from "@nerve/ui";
// helpers
// hooks
import { useMember } from "@/hooks/store/use-member";

type Props = {
  value: string | null;
  onChange: (val: string) => void;
  isDisabled?: boolean;
};

export const MemberSelect = observer(function MemberSelect(props: Props) {
  const { value, onChange, isDisabled = false } = props;
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  // router
  const { projectId } = useParams();
  // store hooks
  const {
    project: { getFilteredProjectMemberIds, getProjectMemberDetails },
  } = useMember();

  // the route's project's members as its members page lists them, but its guests
  const options = (projectId ? getFilteredProjectMemberIds(projectId) : null)
    ?.map((userId) => {
      const memberDetails = projectId ? getProjectMemberDetails(userId, projectId) : null;

      if (!memberDetails?.member) return;
      const isGuest = memberDetails.role === EUserProjectRoles.GUEST;
      if (isGuest) return;

      return {
        value: `${memberDetails?.member.id}`,
        query: `${memberDetails?.member.display_name}`,
        content: (
          <div className="flex items-center gap-2">
            <ProfileAvatar
              alt={memberDetails?.member.display_name}
              fallback={memberDetails?.member.display_name?.[0]?.toUpperCase()}
              source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: memberDetails.member.id }}
              size="xs"
            />
            {memberDetails?.member.display_name}
          </div>
        ),
      };
    })
    .filter((option) => option !== undefined);
  const selectedOption = projectId && value ? getProjectMemberDetails(value, projectId) : null;

  return (
    <CustomSearchSelect
      value={value}
      label={
        <div className="flex h-3.5 items-center gap-2">
          {selectedOption && (
            <ProfileAvatar
              alt={selectedOption.member?.display_name}
              fallback={selectedOption.member?.display_name?.[0]?.toUpperCase()}
              source={{
                kind: "member",
                workspaceID: imageWorkspace?.id ?? "",
                memberID: selectedOption.member?.id ?? "",
              }}
              size="xs"
            />
          )}
          {selectedOption ? (
            selectedOption.member?.display_name
          ) : (
            <div className="flex items-center gap-2">
              <Ban className="h-3.5 w-3.5 rotate-90 text-placeholder" />
              <span className="text-13 text-placeholder">{t("none")}</span>
            </div>
          )}
        </div>
      }
      buttonClassName="!px-3 !py-2 bg-surface-1"
      options={
        options && [
          ...options,
          {
            value: "none",
            query: t("none"),
            content: (
              <div className="flex items-center gap-2">
                <Ban className="h-3.5 w-3.5 rotate-90 text-placeholder" />
                <span className="py-0.5 text-13 text-placeholder">{t("none")}</span>
              </div>
            ),
          },
        ]
      }
      maxHeight="md"
      onChange={onChange}
      disabled={isDisabled}
      searchPlaceholder={t("search")}
      noResultsMessage={t("common.search.no_matches_found")}
      loadingMessage={t("loading")}
    />
  );
});
