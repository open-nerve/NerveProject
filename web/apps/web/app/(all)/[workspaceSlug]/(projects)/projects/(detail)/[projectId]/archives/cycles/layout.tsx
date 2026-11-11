/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { CycleActionsHost } from "@/components/cycles/actions-host";
import { ProjectArchivesHeader } from "../header";
import type { Route } from "./+types/layout";

export default function ProjectArchiveCyclesLayout({ params }: Route.ComponentProps) {
  // the archive's cycles' dialogs and restore outlive the row that asked for them (CycleActionsHost)
  return (
    <CycleActionsHost workspaceSlug={params.workspaceSlug} projectId={params.projectId}>
      <AppHeader
        header={
          <ProjectArchivesHeader workspaceSlug={params.workspaceSlug} projectId={params.projectId} activeTab="cycles" />
        }
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </CycleActionsHost>
  );
}
