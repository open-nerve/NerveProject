/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { ProjectRoot } from "@/components/project/root";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

export const ProjectPageRoot = observer(function ProjectPageRoot() {
  // store
  const { currentWorkspace } = useWorkspace();
  const { fetchArchivedProjects } = useProject();
  useSessionSWR(currentWorkspace && ["ARCHIVED_PROJECTS", currentWorkspace.id, currentWorkspace.slug], (id, slug) =>
    fetchArchivedProjects({ id, slug })
  );

  return <ProjectRoot />;
});
