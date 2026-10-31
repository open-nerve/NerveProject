/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectArchivedIssueDetailsHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectArchivedIssueDetailLayout({ params }: Pick<Route.ComponentProps, "params">) {
  return (
    <Fragment key={JSON.stringify([params.workspaceSlug, params.projectId, params.archivedIssueId])}>
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
    </Fragment>
  );
}
