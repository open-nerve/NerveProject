/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import { observer } from "mobx-react";
import { Outlet, useNavigate } from "react-router";
// hooks
import { useProject } from "@/hooks/store/use-project";
// types
import type { Route } from "./+types/layout";

function ProjectSettingsLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  // router
  const navigate = useNavigate();
  // store hooks
  const { joinedProjectIds } = useProject();

  useEffect(() => {
    if (projectId) return;
    if (joinedProjectIds.length > 0) {
      navigate(`/${workspaceSlug}/settings/projects/${joinedProjectIds[0]}`, { replace: true });
    }
  }, [joinedProjectIds, navigate, workspaceSlug, projectId]);

  return <Outlet />;
}

export default observer(ProjectSettingsLayout);
