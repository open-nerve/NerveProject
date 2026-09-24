/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { GlobalIssuesHeader } from "./header";
import type { Route } from "./+types/layout";

export default function GlobalIssuesLayout({ params }: Route.ComponentProps) {
  return (
    <>
      <AppHeader
        header={<GlobalIssuesHeader workspaceSlug={params.workspaceSlug} globalViewId={params.globalViewId} />}
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}
