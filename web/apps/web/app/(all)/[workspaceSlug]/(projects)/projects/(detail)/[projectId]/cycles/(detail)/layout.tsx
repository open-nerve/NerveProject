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
import { CycleActionsHost } from "@/components/cycles/actions-host";
import { CycleIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectCycleIssuesLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId, cycleId } = params;
  // Keyed by the route's cycle (M3 handoff 1, v0 design 7.7): another cycle, by the header's switcher, a link, a Back or
  // a Forward, mounts its page afresh, so nothing of the cycle before (its reads, a dialog, days picked, a follow-up of
  // nerve's late answer) carries into it. The host of the cycle's dialogs is within the key: they and their follow-ups
  // are the page's, and outlive the header's quick actions, which are under it (CycleActionsHost).
  return (
    <Fragment key={cycleId}>
      <CycleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
        <AppHeader
          header={<CycleIssuesHeader workspaceSlug={workspaceSlug} projectId={projectId} cycleId={cycleId} />}
        />
        <ContentWrapper>
          <Outlet />
        </ContentWrapper>
      </CycleActionsHost>
    </Fragment>
  );
}
