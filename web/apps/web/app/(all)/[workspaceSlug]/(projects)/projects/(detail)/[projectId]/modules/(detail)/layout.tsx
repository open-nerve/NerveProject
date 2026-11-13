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
import { ModuleActionsHost } from "@/components/modules/actions-host";
import { ModuleIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectModuleIssuesLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId, moduleId } = params;
  // Keyed by the route's module (M3 handoff 1, v0 design 7.7): another module, by the header's switcher, a link, a Back
  // or a Forward, mounts its page afresh, so nothing of the module before (its reads, a dialog, days picked, a change
  // still out, the follow-up of nerve's late answer) carries into it. The host of the module's dialogs is within the
  // key: they and their follow-ups are the page's, and outlive the header's quick actions, which are under it
  // (ModuleActionsHost).
  return (
    <Fragment key={moduleId}>
      <ModuleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
        <AppHeader
          header={<ModuleIssuesHeader workspaceSlug={workspaceSlug} projectId={projectId} moduleId={moduleId} />}
        />
        <ContentWrapper>
          <Outlet />
        </ContentWrapper>
      </ModuleActionsHost>
    </Fragment>
  );
}
