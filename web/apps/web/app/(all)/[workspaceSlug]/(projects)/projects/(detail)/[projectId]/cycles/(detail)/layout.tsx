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
import { CycleIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectCycleIssuesLayout({ params }: Route.ComponentProps) {
  // the cycle's dialogs and restore outlive its header's quick actions, which go once it is deleted (CycleActionsHost)
  return (
    <CycleActionsHost workspaceSlug={params.workspaceSlug} projectId={params.projectId}>
      <AppHeader
        header={
          <CycleIssuesHeader
            workspaceSlug={params.workspaceSlug}
            projectId={params.projectId}
            cycleId={params.cycleId}
          />
        }
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </CycleActionsHost>
  );
}
