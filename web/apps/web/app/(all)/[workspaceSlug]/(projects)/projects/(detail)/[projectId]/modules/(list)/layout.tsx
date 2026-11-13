/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ModuleActionsHost } from "@/components/modules/actions-host";
import { ModulesListHeader } from "./header";
import { ModulesListMobileHeader } from "./mobile-header";
import type { Route } from "./+types/layout";

export default function ProjectModulesListLayout({ params }: Route.ComponentProps) {
  // the page's modules' dialogs outlive the row that asked for them (ModuleActionsHost)
  return (
    <ModuleActionsHost workspaceSlug={params.workspaceSlug} projectId={params.projectId}>
      <AppHeader
        header={<ModulesListHeader workspaceSlug={params.workspaceSlug} projectId={params.projectId} />}
        mobileHeader={<ModulesListMobileHeader workspaceSlug={params.workspaceSlug} projectId={params.projectId} />}
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </ModuleActionsHost>
  );
}
