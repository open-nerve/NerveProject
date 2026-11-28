/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ViewActionsHost } from "@/components/views/actions-host";
import { WorkspaceViewsPageStateProvider } from "@/components/workspace/views/page-state";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// local components
import { WorkspaceViewsHeader } from "./header";
import type { Route } from "./+types/layout";

/**
 * The workspace's views list's layout (M7/P6b spec 2.10, R13), of the route's workspace and keyed by it (v0 design
 * 7.7): its header and the page, under the page's state (its search) and the host of the dialogs of the workspace's
 * own views (ViewActionsHost), whose "Add view" and rows' menus open them. Another workspace mounts them afresh, so
 * nothing searched or opened in one carries into the other.
 */
function WorkspaceViewsListLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug } = params;
  // store hooks
  const { getWorkspaceBySlug } = useWorkspace();
  // the route's workspace, as the caller's list gives it
  const workspace = getWorkspaceBySlug(workspaceSlug);

  if (!workspace) return null;

  return (
    <Fragment key={workspaceSlug}>
      <WorkspaceViewsPageStateProvider>
        <ViewActionsHost
          workspaceSlug={workspaceSlug}
          container={{ kind: "workspace", workspaceId: workspace.id, workspaceSlug }}
        >
          <AppHeader header={<WorkspaceViewsHeader workspaceSlug={workspaceSlug} />} />
          <ContentWrapper>
            <Outlet />
          </ContentWrapper>
        </ViewActionsHost>
      </WorkspaceViewsPageStateProvider>
    </Fragment>
  );
}

export default observer(WorkspaceViewsListLayout);
