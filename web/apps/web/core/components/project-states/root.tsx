/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { EUserPermissionsLevel } from "@nerve/constants";
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
  const { groupedProjectStates, fetchProjectStates } = useProjectState();
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

  // Loader
  if (!groupedProjectStates) return <ProjectStateLoader />;

  return <GroupList projectId={projectId} groupedStates={groupedProjectStates} isEditable={isEditable} />;
});
