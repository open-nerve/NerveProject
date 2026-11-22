/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";
// components
import { NotificationPageStateProvider, NotificationsSidebarRoot } from "@/components/workspace-notifications";

/**
 * The notifications page's layout (M7 P4 spec 2.5, 2.6): it holds the page's state (its tab, filters and selection),
 * which ends as the page is left or its workspace changes (the workspace's layout is keyed by the route's slug), and
 * shows the sidebar beside the page's preview.
 */
export default function NotificationsLayout() {
  return (
    <NotificationPageStateProvider>
      <div className="relative flex h-full w-full items-center overflow-hidden">
        <NotificationsSidebarRoot />
        <div className="h-full w-full overflow-hidden overflow-y-auto">
          <Outlet />
        </div>
      </div>
    </NotificationPageStateProvider>
  );
}
