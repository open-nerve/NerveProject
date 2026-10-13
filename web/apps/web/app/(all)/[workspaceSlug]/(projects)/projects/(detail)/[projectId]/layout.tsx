/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet } from "react-router";
// nerve imports
import { Header, Row } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { TabNavigationRoot } from "@/components/navigation/tab-navigation-root";
import { AppSidebarToggleButton } from "@/components/sidebar/sidebar-toggle-button";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
// layouts
import { ProjectAuthWrapper } from "@/layouts/auth-layout/project-wrapper";
// local imports
import type { Route } from "./+types/layout";

function ProjectLayout({ params }: Route.ComponentProps) {
  // router
  const { workspaceSlug, projectId } = params;
  // store hooks
  const { sidebarCollapsed } = useAppTheme();
  // preferences
  const { preferences: projectPreferences } = useProjectNavigationPreferences();

  // keyed by the route's project (v0 design 7.7, state keyed by id): a switch to another project, by a Back or Forward
  // too, mounts its header and its page afresh, so no dialog or state opened in one project (the header's leaving
  // among them, outside the project's wrapper) carries into the other
  return (
    <Fragment key={projectId}>
      {projectPreferences.navigationMode === "TABBED" && (
        <div className="z-20">
          <Row className="flex h-header w-full items-center gap-2 border-b border-subtle bg-surface-1">
            <div className="flex h-full w-full items-center gap-2 divide-x divide-subtle">
              <div className="flex size-full flex-1 items-center gap-2">
                {sidebarCollapsed && (
                  <div className="shrink-0">
                    <AppSidebarToggleButton />
                  </div>
                )}
                <Header className={cn("h-full", { "pl-1.5": !sidebarCollapsed })}>
                  <Header.LeftItem className="flex h-full max-w-full items-center gap-2">
                    <TabNavigationRoot workspaceSlug={workspaceSlug} projectId={projectId} />
                  </Header.LeftItem>
                </Header>
              </div>
            </div>
          </Row>
        </div>
      )}
      <ProjectAuthWrapper projectId={projectId}>
        <Outlet />
      </ProjectAuthWrapper>
    </Fragment>
  );
}

export default observer(ProjectLayout);
