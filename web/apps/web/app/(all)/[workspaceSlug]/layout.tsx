/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";
import { WorkspaceContentWrapper } from "@/components/workspace/content-wrapper";
import { GlobalModals } from "@/components/common/modal/global";
import { WorkspaceAuthWrapper } from "@/layouts/auth-layout/workspace-wrapper";
import type { Route } from "./+types/layout";

export default function WorkspaceLayout({ params }: Route.ComponentProps) {
  return (
    <AuthenticationWrapper>
      <WorkspaceAuthWrapper>
        {/* keyed by the route's workspace (v0 design 7.7, state keyed by id): a switch to another workspace, by a Back
            or Forward too, mounts its header, sidebar, dialogs and page afresh, so no dialog or state opened in one
            workspace carries into the other */}
        <WorkspaceContentWrapper key={params.workspaceSlug}>
          <GlobalModals />
          <Outlet />
        </WorkspaceContentWrapper>
      </WorkspaceAuthWrapper>
    </AuthenticationWrapper>
  );
}
