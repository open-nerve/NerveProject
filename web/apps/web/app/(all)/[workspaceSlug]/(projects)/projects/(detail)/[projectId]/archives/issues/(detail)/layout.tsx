/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectArchivedIssueDetailsHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectArchivedIssueDetailLayout({ params }: Route.ComponentProps) {
  return (
    <>
      <AppHeader
        header={
          <ProjectArchivedIssueDetailsHeader
            workspaceSlug={params.workspaceSlug}
            projectId={params.projectId}
            archivedIssueId={params.archivedIssueId}
          />
        }
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}
