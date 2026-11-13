/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
// hooks
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { PLANNING_PAGE_LAYOUTS } from "../../planning-page-layouts";
import { PlanningPageIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseCalendarRoot } from "../base-calendar-root";

/**
 * The calendar of a planning page: the project page's, with the rows' quick actions of the page's domain, read-only as
 * the domain's list state says (a cycle's: once it has ended or is archived, P5b R5; a module's: once it is archived,
 * P6b R9).
 */
export const PlanningPageCalendarLayout = observer(function PlanningPageCalendarLayout(props: {
  storeType: PlanningPageStoreType;
}) {
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { allowPermissions } = useUserPermissions();
  const { useListState } = PLANNING_PAGE_LAYOUTS[props.storeType];
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
    <BaseCalendarRoot
      QuickActions={PlanningPageIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
      readOnly={readOnly}
    />
  );
});
