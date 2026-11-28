/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { ProjectIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";

/**
 * A workspace's view's page's table (M7/P6b spec 2.8, R7): the workspace's list, its rows of the caller's projects.
 * Each row is changed in its own project, while the caller is that project's admin or member (his role there, by the
 * route's workspace), and its quick actions are the project page's (ProjectIssueQuickActions, as a person's page's
 * rows). The route names no project: the table creates nothing (no quick add) and shows no cycle or modules column.
 */
export const WorkspaceSpreadsheetRoot = observer(function WorkspaceSpreadsheetRoot(props: { workspaceSlug: string }) {
  const { workspaceSlug } = props;
  // store hooks
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // the row's project's role
  const canEditPropertiesBasedOnProject = (projectId: string) => {
    const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
    return role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  };

  return (
    <BaseSpreadsheetRoot
      QuickActions={ProjectIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
    />
  );
});
