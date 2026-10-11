/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { renderFormattedDate } from "@nerve/utils";
// components
import { MemberHeaderColumn } from "@/components/project/member-header-column";
import {
  canRemove,
  roleChoices,
  type MembershipCaller,
  type ShownMembership,
} from "@/components/project/project-roles";
import { AccountTypeColumn, NameColumn } from "@/components/project/settings/member-columns";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
import type { IMemberFilters } from "@/store/member/utils";

type TUseProjectColumnsProps = {
  projectId: string;
  workspaceSlug: string;
};

export const useProjectColumns = (props: TUseProjectColumnsProps) => {
  const { projectId, workspaceSlug } = props;
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState<IProjectMemberDetails | null>(null);

  // store hooks
  const { data: currentUser } = useUser();
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { getProjectById } = useProject();
  const {
    project: {
      filters: { getFilters, updateFilters },
    },
    workspace: { getWorkspaceMemberDetails },
  } = useMember();
  // derived values: the caller and each membership, as nerve decides a change of a membership (M3 design 3.5)
  const caller: MembershipCaller = {
    workspaceRole: getWorkspaceRoleByWorkspaceSlug(workspaceSlug),
    projectRole: getProjectById(projectId)?.member_role,
  };
  const shown = (rowData: IProjectMemberDetails): ShownMembership => ({
    own: rowData.member.id === currentUser?.id,
    role: rowData.role,
    workspaceRole: getWorkspaceMemberDetails(rowData.member.id)?.role,
  });

  const displayFilters = getFilters(projectId);

  // handlers
  const handleDisplayFilterUpdate = (filters: Partial<IMemberFilters>) => {
    updateFilters(projectId, filters);
  };

  const columns = [
    {
      key: "Full Name",
      content: "Full name",
      thClassName: "text-left",
      thRender: () => (
        <MemberHeaderColumn
          property="full_name"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: IProjectMemberDetails) => (
        <NameColumn
          rowData={rowData}
          workspaceSlug={workspaceSlug}
          own={shown(rowData).own}
          removable={canRemove(caller, shown(rowData))}
          setRemoveMemberModal={setRemoveMemberModal}
        />
      ),
    },
    {
      key: "Display Name",
      content: "Display name",
      thRender: () => (
        <MemberHeaderColumn
          property="display_name"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: IProjectMemberDetails) => <div className="w-32">{rowData.member.display_name}</div>,
    },
    {
      key: "Email",
      content: "Email",
      thRender: () => (
        <MemberHeaderColumn
          property="email"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: IProjectMemberDetails) => <div className="w-48 text-secondary">{rowData.member.email}</div>,
    },
    {
      key: "Account Type",
      content: "Account type",
      thRender: () => (
        <MemberHeaderColumn
          property="role"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: IProjectMemberDetails) => (
        <AccountTypeColumn
          rowData={rowData}
          choices={roleChoices(caller, shown(rowData))}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
        />
      ),
    },
    {
      key: "Joining Date",
      content: "Joining date",
      thRender: () => (
        <MemberHeaderColumn
          property="joining_date"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: IProjectMemberDetails) => <div>{renderFormattedDate(rowData.created_at)}</div>,
    },
  ];
  return {
    columns,
    removeMemberModal,
    setRemoveMemberModal,
    displayFilters,
    handleDisplayFilterUpdate,
  };
};
