/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
import { useCycleListState } from "@/hooks/use-cycle-list-state";
// local imports
import { CycleIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";

/** The table of a cycle's page: the project page's, read-only once the cycle has ended or is archived (R5). */
export const CycleSpreadsheetLayout = observer(function CycleSpreadsheetLayout() {
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { allowPermissions } = useUserPermissions();
  const { readOnly } = useCycleListState();

  if (!workspaceSlug) return null;

  const canEditPropertiesBasedOnProject = (projectId: string) =>
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );

  return (
    <BaseSpreadsheetRoot
      QuickActions={CycleIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
      readOnly={readOnly}
    />
  );
});
