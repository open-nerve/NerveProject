/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { IntakePageHeader } from "@/components/inbox/page-header";
import { IntakePageStateProvider } from "@/components/inbox/page-state";

/**
 * A project's intake layout (M7 P5a spec 2.6, 2.7): the intake's header, and the page under the page's state (the
 * filters of the tab shown), which ends as the page is left or its project or workspace changes (their layouts are
 * keyed by their ids).
 */
export default function ProjectInboxIssuesLayout() {
  return (
    <>
      <AppHeader header={<IntakePageHeader />} />
      <ContentWrapper>
        <IntakePageStateProvider>
          <Outlet />
        </IntakePageStateProvider>
      </ContentWrapper>
    </>
  );
}
