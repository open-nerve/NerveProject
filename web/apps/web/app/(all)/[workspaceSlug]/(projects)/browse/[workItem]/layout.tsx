/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { Outlet } from "react-router";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectWorkItemDetailsHeader } from "./header";
import type { Route } from "./+types/layout";

export default function ProjectIssueDetailsLayout({ params }: Route.ComponentProps) {
  return (
    <>
      <ProjectWorkItemDetailsHeader workspaceSlug={params.workspaceSlug} workItem={params.workItem} />
      <ContentWrapper className="overflow-hidden">
        <Outlet />
      </ContentWrapper>
    </>
  );
}
