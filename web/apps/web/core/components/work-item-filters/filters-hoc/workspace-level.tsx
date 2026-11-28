/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { CreateUpdateViewModal } from "@/components/views/modal";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { joinedAndUnarchived } from "@/lib/project-scope";
// local imports
import { WorkItemFiltersHOC } from "./base";
import type { TSharedWorkItemFiltersHOCProps } from "./shared";
import type { TWorkspaceViewSavingProps } from "./use-workspace-view-saving";
import { useWorkspaceViewSaving } from "./use-workspace-view-saving";

type TWorkspaceLevelWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & {
  workspaceSlug: string;
} & TWorkspaceViewSavingProps;

/**
 * The filters of a workspace's list page (a workspace's view's, the profile's work items; M7/P6b spec 2.4, 2.9), with
 * the workspace's options: its members; its projects the caller is a member of, unarchived; their labels, as the label
 * store holds them (all of them once the page has read the workspace's catalog); their states, each project's as the
 * state store gives them. Save view and Update view as the page offers them (useWorkspaceViewSaving): Save view opens
 * the view's dialog here, a view of the workspace's own, with what the new view keeps. The profile offers neither.
 */
export const WorkspaceLevelWorkItemFiltersHOC = observer(function WorkspaceLevelWorkItemFiltersHOC(
  props: TWorkspaceLevelWorkItemFiltersHOCProps
) {
  const { enableSaveView, enableUpdateView, ...filters } = props;
  const { children, entityId, initialWorkItemFilters, workspaceSlug } = filters;
  // hooks
  const { knownProjectIds, getProjectById } = useProject();
  const { getWorkspaceBySlug } = useWorkspace();
  const { labelMap } = useLabel();
  const { getProjectStateIds } = useProjectState();
  const {
    workspace: { getWorkspaceMemberIds },
  } = useMember();
  const saving = useWorkspaceViewSaving({
    workspaceSlug,
    entityId: entityId ?? "",
    shown: initialWorkItemFilters,
    enableSaveView,
    enableUpdateView,
  });
  // derived values: the route's workspace, and the caller's projects of it
  const workspace = getWorkspaceBySlug(workspaceSlug);
  const projectIds = workspace
    ? knownProjectIds(workspace.id).filter((id) => {
        const project = getProjectById(id);
        return !!project && project.workspace_id === workspace.id && joinedAndUnarchived(project);
      })
    : [];
  const labelIds = Object.values(labelMap)
    .filter((label) => projectIds.includes(label.project_id))
    .map((label) => label.id);
  const stateIds = projectIds.flatMap((id) => getProjectStateIds(id) ?? []);

  return (
    <>
      {workspace && (
        <CreateUpdateViewModal
          workspaceSlug={workspaceSlug}
          container={{ kind: "workspace", workspaceId: workspace.id, workspaceSlug }}
          content={saving.saveAs ?? undefined}
          isOpen={saving.saveAs !== null}
          handleClose={saving.closeSaveAs}
        />
      )}
      <WorkItemFiltersHOC
        {...filters}
        memberIds={getWorkspaceMemberIds(workspaceSlug)}
        projectIds={projectIds}
        labelIds={labelIds}
        stateIds={stateIds}
        saveViewOptions={saving.saveViewOptions}
        updateViewOptions={saving.updateViewOptions}
      >
        {children}
      </WorkItemFiltersHOC>
    </>
  );
});
