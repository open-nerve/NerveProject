/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { matchPath } from "react-router";
import type { TIssue } from "@nerve/types";
import type { TNavigationItem } from "@/components/navigation/tab-navigation-root";

type UseActiveTabProps = {
  navigationItems: TNavigationItem[];
  pathname: string;
  workItemId?: string;
  workItem?: TIssue;
  projectId: string;
};

export const useActiveTab = ({ navigationItems, pathname, workItemId, workItem, projectId }: UseActiveTabProps) => {
  // Check if a navigation item is active
  const isActive = useCallback(
    (item: TNavigationItem) => {
      // Work item condition
      const workItemCondition = workItemId && workItem && workItem?.project_id === projectId;
      // Is active
      const isWorkItemActive = item.key === "work_items" && workItemCondition;
      // Pathname condition: the item's address or anything below it
      const isPathnameActive = matchPath({ path: item.href, end: false }, pathname) !== null;
      // Return
      return isWorkItemActive || isPathnameActive;
    },
    [pathname, workItem, workItemId, projectId]
  );

  // Find active item
  const activeItem = useMemo(() => navigationItems.find((item) => isActive(item)), [navigationItems, isActive]);

  return { isActive, activeItem };
};
