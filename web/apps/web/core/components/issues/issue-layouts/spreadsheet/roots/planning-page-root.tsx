/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { PLANNING_PAGE_LAYOUTS, type PlanningPageLayoutStoreType } from "../../planning-page-layouts";
import { BaseSpreadsheetRoot } from "../base-spreadsheet-root";

/**
 * The table of a planning page: the project page's, with the rows' quick actions of the page's domain, read-only as
 * the domain's list state says (a cycle's: once it has ended or is archived, P5b R5).
 */
export const PlanningPageSpreadsheetLayout = observer(function PlanningPageSpreadsheetLayout(props: {
  storeType: PlanningPageLayoutStoreType;
}) {
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { allowPermissions } = useUserPermissions();
  const { useListState, QuickActions } = PLANNING_PAGE_LAYOUTS[props.storeType];
  const { readOnly } = useListState();

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
      QuickActions={QuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
      readOnly={readOnly}
    />
  );
});
