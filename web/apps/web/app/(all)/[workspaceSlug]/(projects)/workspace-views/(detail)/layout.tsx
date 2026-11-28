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
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// local components
import { WorkspaceViewHeader } from "./[globalViewId]/header";
import type { Route } from "./+types/layout";

/**
 * A workspace's view's page's layout (M7/P6b spec 2.8, R11; M3 handoff 1, v0 design 7.7), of the route's workspace and
 * view, keyed by the view (the workspace's own layout is keyed by the workspace): another view, by the header's
 * switcher, a link, a Back or a Forward, mounts its page afresh, so nothing of the view before (its read, its draft, a
 * dialog, a follow-up of nerve's late answer) carries into it. The host of the workspace's views' dialogs is within the
 * key: they and their follow-ups are the page's, and outlive the header's quick actions, which are under it.
 */
function WorkspaceViewLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, globalViewId } = params;
  // store hooks
  const { getWorkspaceBySlug } = useWorkspace();
  // the route's workspace, as the caller's list gives it
  const workspace = getWorkspaceBySlug(workspaceSlug);

  if (!workspace) return null;

  return (
    <Fragment key={globalViewId}>
      <ViewActionsHost
        workspaceSlug={workspaceSlug}
        container={{ kind: "workspace", workspaceId: workspace.id, workspaceSlug }}
      >
        <AppHeader header={<WorkspaceViewHeader workspaceSlug={workspaceSlug} globalViewId={globalViewId} />} />
        <ContentWrapper>
          <Outlet />
        </ContentWrapper>
      </ViewActionsHost>
    </Fragment>
  );
}

export default observer(WorkspaceViewLayout);
