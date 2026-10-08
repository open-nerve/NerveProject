/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { observer } from "mobx-react";
// components
import { EUserPermissionsLevel } from "@nerve/constants";
import type { StateCreate } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { EUserProjectRoles } from "@nerve/types";
import { ProjectStateLoader, GroupList } from "@/components/project-states";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
import { useUserPermissions } from "@/hooks/store/user";
import { useSessionSWR } from "@/lib/use-session-swr";

type TProjectState = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectStateRoot = observer(function ProjectStateRoot(props: TProjectState) {
  const { workspaceSlug, projectId } = props;
  // hooks
  const {
    groupedProjectStates,
    fetchProjectStates,
    createState,
    updateState,
    moveState,
    deleteState,
    markStateAsDefault,
  } = useProjectState();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const isEditable = allowPermissions(
    [EUserProjectRoles.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  // Fetching all project states
  useSessionSWR(["PROJECT_STATES", projectId], (id) => fetchProjectStates(id));

  // State operations callbacks
  const stateOperationsCallbacks: TStateOperationsCallbacks = useMemo(
    () => ({
      createState: (data: StateCreate) => createState(projectId, data),
      updateState,
      moveState,
      deleteState,
      markStateAsDefault,
    }),
    [projectId, createState, updateState, moveState, deleteState, markStateAsDefault]
  );

  // Loader
  if (!groupedProjectStates) return <ProjectStateLoader />;

  return (
    <GroupList
      groupedStates={groupedProjectStates}
      stateOperationsCallbacks={stateOperationsCallbacks}
      isEditable={isEditable}
    />
  );
});
