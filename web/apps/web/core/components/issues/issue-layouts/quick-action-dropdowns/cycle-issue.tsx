/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import { useUserPermissions } from "@/hooks/store/user";
import { useCycleListState } from "@/hooks/use-cycle-list-state";
import { useCycleRemovalRequest } from "../command-host";
import type { IQuickActionProps } from "../list/list-view-types";
import { WorkItemQuickActions } from "./project-issue";

/**
 * A row's quick actions on a cycle's page (P5b spec 2.6, R6; re-66 L4): the project page's, whose edit and copy dialogs
 * are the cycle page's (a copy joins the cycle while it takes work items), and the removal of the work item from the
 * cycle. That is offered to whoever may change the work item in its project, while the row is not read-only and the
 * cycle takes work items (useCycleListState: its project's cycles on, the cycle neither ended nor archived). It sends
 * the work item's change to no cycle through the page's list, whose layout says once nerve has confirmed it (the row
 * then off the list) or why nerve refused it.
 */
export const CycleIssueQuickActions = observer(function CycleIssueQuickActions(props: IQuickActionProps) {
  const { issue, handleUpdate, readOnly = false } = props;
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { allowPermissions } = useUserPermissions();
  const { joinable } = useCycleListState();
  const requestRemoval = useCycleRemovalRequest();
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
  const handleRemoveFromCycle =
    removable && handleUpdate && requestRemoval
      ? () => void requestRemoval(issue, () => handleUpdate({ cycle_id: null })).catch(() => undefined)
      : undefined;

  return (
    <WorkItemQuickActions {...props} storeType={EIssuesStoreType.CYCLE} handleRemoveFromCycle={handleRemoveFromCycle} />
  );
});
