/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "react-router";
// components
import { ProjectRoot } from "@/components/project/root";
// local imports
import { useArchivedProjectsFetch } from "./use-archived-projects-fetch";

export const ProjectPageRoot = observer(function ProjectPageRoot() {
  const { workspaceSlug } = useParams();
  useArchivedProjectsFetch(workspaceSlug);

  return <ProjectRoot />;
});
