/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { EIssuesStoreType } from "@nerve/types";
import { isPlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useRemovalFromViewRequest } from "../command-host";
import type { IQuickActionProps } from "../list/list-view-types";
import { PLANNING_PAGE_LAYOUTS } from "../planning-page-layouts";
import { WorkItemQuickActions } from "./project-issue";

/** The page's store type and its domain's row: a cycle's or a module's page lists the rows whose actions these are. */
function planningPageOf(storeType: EIssuesStoreType) {
  if (!isPlanningPageStoreType(storeType)) throw new Error(`No cycle's or module's page lists ${storeType} rows`);
  return { storeType, ...PLANNING_PAGE_LAYOUTS[storeType] };
}

/**
 * A row's quick actions on a cycle's or a module's page (P5b spec 2.6, R6; P6b spec 2.2, 2.7, R5, R7; re-66 L4): the
 * project page's, whose edit and copy dialogs are the page's (a copy joins its cycle or module while it takes work
 * items), and the removal of the work item from the page's cycle or module, by the label of its domain. That is offered
 * to whoever may change the work item in its project, while the row is not read-only and the page's cycle or module
 * takes work items (its domain's list state). It goes through the page's list as its domain sends it
 * (handleRemoveFromView), one at a time for the work item, and the layout's command host says once nerve has confirmed
 * it (the row then off the list), as the domain words it, or why nerve refused it. The page's store type, which its
 * layout root provides, picks the domain's row of the table; nothing here branches on it.
 */
export const PlanningPageIssueQuickActions = observer(function PlanningPageIssueQuickActions(props: IQuickActionProps) {
  const { issue, handleRemoveFromView, readOnly = false } = props;
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { storeType, useListState, remove } = planningPageOf(useIssueStoreType());
  const { allowPermissions } = useUserPermissions();
  const { joinable } = useListState();
  const requestRemoval = useRemovalFromViewRequest();
  const removable =
    !readOnly &&
    joinable &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      issue.project_id
    );
  // the layout says what came of it; a refusal is no failure of the menu's
  const removeFromView =
    removable && handleRemoveFromView && requestRemoval
      ? {
          label: remove.label,
          remove: () => void requestRemoval(issue, handleRemoveFromView, remove.success).catch(() => undefined),
        }
      : undefined;

  return <WorkItemQuickActions {...props} storeType={storeType} removeFromView={removeFromView} />;
});
