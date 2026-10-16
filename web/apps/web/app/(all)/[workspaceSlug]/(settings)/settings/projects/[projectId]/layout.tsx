/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet, useLocation } from "react-router";
// components
import { getProjectActivePath } from "@/components/settings/helper";
import { SettingsMobileNav } from "@/components/settings/mobile/nav";
// layouts
import { ProjectAuthWrapper } from "@/layouts/auth-layout/project-wrapper";
// types
import type { Route } from "./+types/layout";
import { ProjectSettingsSidebarRoot } from "@/components/settings/project/sidebar";

function ProjectDetailSettingsLayout({ params }: Route.ComponentProps) {
  const { projectId } = params;
  // router
  const { pathname } = useLocation();

  // keyed by the route's project (v0 design 7.7, state keyed by id): a switch to another project's settings, by a Back
  // or Forward too, mounts its sidebar and its page afresh, so no dialog or state opened in one project carries into
  // the other
  return (
    <Fragment key={projectId}>
      <SettingsMobileNav
        hamburgerContent={(props) => <ProjectSettingsSidebarRoot {...props} projectId={projectId} />}
        activePath={getProjectActivePath(pathname) || ""}
      />
      <div className="inset-y-0 flex h-full w-full flex-row">
        <div className="relative flex size-full">
          <div className="hidden h-full shrink-0 md:block">
            <ProjectSettingsSidebarRoot projectId={projectId} />
          </div>
          <ProjectAuthWrapper projectId={projectId}>
            <Outlet />
          </ProjectAuthWrapper>
        </div>
      </div>
    </Fragment>
  );
}

export default observer(ProjectDetailSettingsLayout);
