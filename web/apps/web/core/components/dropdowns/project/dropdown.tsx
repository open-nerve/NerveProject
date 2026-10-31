/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { ProjectDropdownBase, type ProjectDropdownProps } from "./base";

export const ProjectDropdown = observer(function ProjectDropdown(props: ProjectDropdownProps) {
  // store hooks
  const { joinedProjectIds, getProjectById } = useProject();

  return <ProjectDropdownBase {...props} getProjectById={getProjectById} projectIds={joinedProjectIds} />;
});
