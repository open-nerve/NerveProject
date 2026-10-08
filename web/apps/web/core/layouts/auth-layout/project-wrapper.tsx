/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";
import { observer } from "mobx-react";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { ProjectAccessRestriction } from "@/components/auth-screens/project/project-access-restriction";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { useProjectFetch } from "./use-project-fetch";

interface IProjectAuthWrapper {
  projectId: string;
  children: ReactNode;
}

export const ProjectAuthWrapper = observer(function ProjectAuthWrapper(props: IProjectAuthWrapper) {
  const { projectId, children } = props;
  // states
  const [isJoiningProject, setIsJoiningProject] = useState(false);
  // store hooks
  const { joinProject } = useProject();

  // the project side of what every page of a project fetches (M3 design 7.1), and what nerve's read of the project
  // decides it is to the caller (3.19)
  const access = useProjectFetch(projectId);

  // handle join project
  const handleJoinProject = () => {
    setIsJoiningProject(true);
    joinProject(projectId).finally(() => setIsJoiningProject(false));
  };

  // nerve's read of the project has not answered yet
  if (access.kind === "loading") return null;

  // nerve could not be reached: the page says so, and tries again when asked (M2 design 7.1)
  if (access.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={access.retry} />;

  // a project the caller sees and is no member of, which he may join; or one not found to him
  if (access.kind !== "member") {
    return (
      <ProjectAccessRestriction
        canJoin={access.kind === "not-member"}
        handleJoinProject={handleJoinProject}
        isJoinButtonDisabled={isJoiningProject}
      />
    );
  }

  return <>{children}</>;
});
