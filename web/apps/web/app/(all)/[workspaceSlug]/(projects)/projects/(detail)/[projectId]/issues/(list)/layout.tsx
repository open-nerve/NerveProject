/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { Outlet } from "react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectIssuesHeader } from "./header";
import { ProjectIssuesMobileHeader } from "./mobile-header";
import type { Route } from "./+types/layout";

export default function ProjectIssuesLayout({ params }: Route.ComponentProps) {
  // the small screen's header says itself when it shows: it is an observer, this layout is not (re-76 F1)
  return (
    <>
      <AppHeader
        header={<ProjectIssuesHeader />}
        mobileHeader={<ProjectIssuesMobileHeader projectId={params.projectId} />}
      />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}
