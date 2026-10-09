/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useParams } from "react-router";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedDate } from "@nerve/utils";
import { MemberHeaderColumn } from "@/components/project/member-header-column";
import type { WorkspaceMember } from "@nerve/api-client";
import { AccountTypeColumn, NameColumn } from "@/components/workspace/settings/member-columns";
import { useMember } from "@/hooks/store/use-member";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import type { IMemberFilters } from "@/store/member/utils";

export const useMemberColumns = () => {
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState<WorkspaceMember | null>(null);

  const { workspaceSlug } = useParams();

  const { data: currentUser } = useUser();
  const { allowPermissions } = useUserPermissions();
  const {
    workspace: {
      filtersStore: { filters, updateFilters },
    },
  } = useMember();
  const { t } = useTranslation();

  // derived values
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);

  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const isSuspended = (rowData: WorkspaceMember) => rowData.is_active === false;

  // handlers
  const handleDisplayFilterUpdate = (filterUpdates: Partial<IMemberFilters>) => {
    updateFilters(filterUpdates);
  };

  const columns = [
    {
      key: "Full name",
      content: t("workspace_settings.settings.members.details.full_name"),
      thClassName: "text-left",
      thRender: () => (
        <MemberHeaderColumn
          property="full_name"
          displayFilters={filters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: WorkspaceMember) =>
        workspaceSlug && (
          <NameColumn
            rowData={rowData}
            workspaceSlug={workspaceSlug}
            isAdmin={isAdmin}
            currentUser={currentUser}
            setRemoveMemberModal={setRemoveMemberModal}
          />
        ),
    },

    {
      key: "Display name",
      content: t("workspace_settings.settings.members.details.display_name"),
      tdRender: (rowData: WorkspaceMember) => (
        <div className={`w-32 ${isSuspended(rowData) ? "text-placeholder" : ""}`}>{rowData.member.display_name}</div>
      ),
      thRender: () => (
        <MemberHeaderColumn
          property="display_name"
          displayFilters={filters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
    },

    {
      key: "Email address",
      content: t("workspace_settings.settings.members.details.email_address"),
      tdRender: (rowData: WorkspaceMember) => (
        <div className={`w-48 truncate ${isSuspended(rowData) ? "text-placeholder" : ""}`}>{rowData.member.email}</div>
      ),
      thRender: () => (
        <MemberHeaderColumn
          property="email"
          displayFilters={filters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
    },

    {
      key: "Account type",
      content: t("workspace_settings.settings.members.details.account_type"),
      thRender: () => (
        <MemberHeaderColumn
          property="role"
          displayFilters={filters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: WorkspaceMember) =>
        workspaceSlug && <AccountTypeColumn rowData={rowData} workspaceSlug={workspaceSlug} />,
    },

    {
      key: "Joining date",
      content: t("workspace_settings.settings.members.details.joining_date"),
      tdRender: (rowData: WorkspaceMember) =>
        isSuspended(rowData) ? null : <div>{renderFormattedDate(rowData.created_at)}</div>,
      thRender: () => (
        <MemberHeaderColumn
          property="joining_date"
          displayFilters={filters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
    },
  ];
  return { columns, removeMemberModal, setRemoveMemberModal };
};
