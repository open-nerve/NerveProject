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
// local components
import { ProjectViewIssuesHeader } from "./[viewId]/header";
import { ProjectViewMobileHeader } from "./[viewId]/mobile-header";
import type { Route } from "./+types/layout";

export default function ProjectViewIssuesLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId, viewId } = params;
  // Keyed by the route's view (M3 handoff 1, v0 design 7.7; P6a spec 2.6): another view, by the header's switcher, a
  // link, a Back or a Forward, mounts its page afresh, so nothing of the view before (its read, its draft, a dialog, a
  // follow-up of nerve's late answer) carries into it. The host of the view's dialogs is within the key: they and their
  // follow-ups are the page's, and outlive the header's quick actions, which are under it (ViewActionsHost). The small
  // screen's header says itself when it shows: it is an observer, this layout is not (re-76 F1).
  return (
    <Fragment key={viewId}>
      <ViewActionsHost workspaceSlug={workspaceSlug} container={{ kind: "project", projectId }}>
        <AppHeader
          header={<ProjectViewIssuesHeader workspaceSlug={workspaceSlug} projectId={projectId} viewId={viewId} />}
          mobileHeader={<ProjectViewMobileHeader workspaceSlug={workspaceSlug} projectId={projectId} viewId={viewId} />}
        />
        <ContentWrapper>
          <Outlet />
        </ContentWrapper>
      </ViewActionsHost>
    </Fragment>
  );
}
