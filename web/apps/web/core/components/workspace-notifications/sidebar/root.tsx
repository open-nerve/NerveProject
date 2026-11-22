/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Row } from "@nerve/ui";
import { cn } from "@nerve/utils";
// store
import { filterKey } from "@/store/notification/filters";
// local imports
import { useNotificationPageState } from "../page-state";
import { NotificationSidebarHeader } from "./header";
import { NotificationList } from "./list";
import { NotificationTabs } from "./tabs";

/**
 * The notifications page's sidebar (P4 spec 2.6): its head, its tabs and the chosen tab's list, one list for each set
 * of filters. On a narrow screen it gives way to the preview once a notification is selected.
 */
export const NotificationsSidebarRoot = observer(function NotificationsSidebarRoot() {
  const { filters, selected } = useNotificationPageState();
  return (
    <div
      className={cn(
        "relative z-[10] h-full flex-shrink-0 border-0 border-subtle bg-surface-1 transition-all max-md:overflow-hidden md:border-r",
        selected ? "w-0 md:w-3/12" : "w-full md:w-3/12"
      )}
    >
      <div className="relative flex h-full w-full flex-col">
        <Row className="flex h-header flex-shrink-0 border-b border-subtle">
          <NotificationSidebarHeader />
        </Row>
        <NotificationTabs>
          <NotificationList key={filterKey(filters)} />
        </NotificationTabs>
      </div>
    </div>
  );
});
