/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { CreateUpdateProjectViewModal } from "@/components/views/modal";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useProjectState } from "@/hooks/store/use-project-state";
// local imports
import { WorkItemFiltersHOC } from "./base";
import type { TSharedWorkItemFiltersHOCProps } from "./shared";
import type { TProjectViewSavingProps } from "./use-project-view-saving";
import { useProjectViewSaving } from "./use-project-view-saving";

type TProjectLevelWorkItemFiltersHOCProps = TSharedWorkItemFiltersHOCProps & {
  workspaceSlug: string;
  projectId: string;
} & TProjectViewSavingProps;

/**
 * The filters of a project's list page (its work items, a cycle's, a module's, a view's, the archive's), with the
 * project's labels, members and states as their options, and the page's Save view and Update view as the page offers
 * them (useProjectViewSaving, P6a spec 2.7): Save view opens the view's dialog here, with what the new view keeps.
 */
export const ProjectLevelWorkItemFiltersHOC = observer(function ProjectLevelWorkItemFiltersHOC(
  props: TProjectLevelWorkItemFiltersHOCProps
) {
  const { enableSaveView, enableUpdateView, preferences, viewTerms, ...filters } = props;
  const { children, entityId, initialWorkItemFilters, projectId, workspaceSlug } = filters;
  // hooks
  const { getProjectLabelIds } = useLabel();
  const {
    project: { getProjectMemberIds },
  } = useMember();
  const { getProjectStateIds } = useProjectState();
  const saving = useProjectViewSaving({
    workspaceSlug,
    projectId,
    entityId: entityId ?? "",
    shown: initialWorkItemFilters,
    enableSaveView,
    enableUpdateView,
    preferences,
    viewTerms,
  });

  return (
    <>
      <CreateUpdateProjectViewModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        content={saving.saveAs ?? undefined}
        isOpen={saving.saveAs !== null}
        handleClose={saving.closeSaveAs}
      />
      <WorkItemFiltersHOC
        {...filters}
        labelIds={getProjectLabelIds(projectId)}
        memberIds={getProjectMemberIds(projectId, false) ?? undefined}
        stateIds={getProjectStateIds(projectId)}
        saveViewOptions={saving.saveViewOptions}
        updateViewOptions={saving.updateViewOptions}
      >
        {children}
      </WorkItemFiltersHOC>
    </>
  );
});
