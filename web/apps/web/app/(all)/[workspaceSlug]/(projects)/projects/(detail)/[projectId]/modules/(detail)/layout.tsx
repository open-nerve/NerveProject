/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { Outlet } from "react-router";
// nerve imports
import { EIssuesStoreType } from "@nerve/types";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ModuleActionsHost } from "@/components/modules/actions-host";
import { PlanningPageMobileHeader } from "@/components/issues/issue-layouts/planning-page-mobile-header";
import { ModuleIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectModuleIssuesLayout({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId, moduleId } = params;
  // Keyed by the route's module (M3 handoff 1, v0 design 7.7): another module, by the header's switcher, a link, a Back
  // or a Forward, mounts its page afresh, so nothing of the module before (its reads, a dialog, days picked, a change
  // still out, the follow-up of nerve's late answer) carries into it. The host of the module's dialogs is within the
  // key: they and their follow-ups are the page's, and outlive the header's quick actions, which are under it
  // (ModuleActionsHost). The small screen's header says itself when it shows: it is an observer, this layout is not
  // (re-76 F1).
  return (
    <Fragment key={moduleId}>
      <ModuleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
        <AppHeader
          header={<ModuleIssuesHeader workspaceSlug={workspaceSlug} projectId={projectId} moduleId={moduleId} />}
          mobileHeader={
            <PlanningPageMobileHeader
              storeType={EIssuesStoreType.MODULE}
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              entityId={moduleId}
            />
          }
        />
        <ContentWrapper>
          <Outlet />
        </ContentWrapper>
      </ModuleActionsHost>
    </Fragment>
  );
}
