/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ModuleIssuesHeader } from "./header";
import { ModuleIssuesMobileHeader } from "./mobile-header";
import type { Route } from "./+types/layout";

export default function ProjectModuleIssuesLayout({ params }: Route.ComponentProps) {
  return (
    <>
      <AppHeader
        header={
          <ModuleIssuesHeader
            workspaceSlug={params.workspaceSlug}
            projectId={params.projectId}
            moduleId={params.moduleId}
          />
        }
        mobileHeader={
          <ModuleIssuesMobileHeader
            workspaceSlug={params.workspaceSlug}
            projectId={params.projectId}
            moduleId={params.moduleId}
          />
        }
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}
