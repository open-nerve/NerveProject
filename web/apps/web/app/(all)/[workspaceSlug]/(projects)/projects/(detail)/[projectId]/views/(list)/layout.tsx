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
import { ViewActionsHost } from "@/components/views/actions-host";
import { ViewsPageStateProvider } from "@/components/views/page-state";
// local components
import { ProjectViewsHeader } from "./header";
import { ViewMobileHeader } from "./mobile-header";
import type { Route } from "./+types/layout";

/**
 * A project's views list's layout (P6a spec 2.3, R5, R17), of the route's project and keyed by it (M3 handoff §2, v0
 * design 7.7): its header and its small screen's, and the page, under the page's state (the search, the filters and the
 * order chosen) and the host of the dialogs its views' quick actions open, which outlive the row that asked for them
 * (ViewActionsHost). Another project, by a link, a Back or a Forward, mounts them afresh, so nothing chosen or opened
 * in one project carries into the other.
 */
export default function ProjectViewsListLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  return (
    <Fragment key={projectId}>
      <ViewsPageStateProvider>
        <ViewActionsHost workspaceSlug={workspaceSlug} container={{ kind: "project", projectId }}>
          <AppHeader
            header={<ProjectViewsHeader workspaceSlug={workspaceSlug} projectId={projectId} />}
            mobileHeader={<ViewMobileHeader workspaceSlug={workspaceSlug} projectId={projectId} />}
          />
          <ContentWrapper>
            <Outlet />
          </ContentWrapper>
        </ViewActionsHost>
      </ViewsPageStateProvider>
    </Fragment>
  );
}
