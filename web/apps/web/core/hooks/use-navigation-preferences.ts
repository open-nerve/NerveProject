/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { TProjectNavigationPreferences, TProjectNavigationMode } from "@nerve/types";
import { DEFAULT_PROJECT_PREFERENCES } from "@nerve/types";
import { useWorkspace } from "./store/use-workspace";

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams();
  const { getProjectNavigationPreferences, updateProjectNavigationPreferences } = useWorkspace();

  // Get preferences from the store
  const storePreferences = getProjectNavigationPreferences(workspaceSlug || "");

  // Computed preferences with fallback logic: API → defaults
  const preferences: TProjectNavigationPreferences = useMemo(() => {
    // 1. Try API data first
    if (
      storePreferences &&
      (storePreferences.navigation_control_preference || storePreferences.navigation_project_limit !== undefined)
    ) {
      const limit = storePreferences.navigation_project_limit ?? DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount;

      return {
        navigationMode: storePreferences.navigation_control_preference || DEFAULT_PROJECT_PREFERENCES.navigationMode,
        limitedProjectsCount: limit > 0 ? limit : DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount,
        showLimitedProjects: limit > 0, // Derived: 0 = false, >0 = true
      };
    }

    // 2. Fall back to defaults
    return DEFAULT_PROJECT_PREFERENCES;
  }, [storePreferences]);

  // Update navigation mode
  const updateNavigationMode = useCallback(
    async (mode: TProjectNavigationMode) => {
      if (!workspaceSlug) return;

      await updateProjectNavigationPreferences(workspaceSlug, {
        navigation_control_preference: mode,
      });
    },
    [workspaceSlug, updateProjectNavigationPreferences]
  );

  // Update show limited projects
  const updateShowLimitedProjects = useCallback(
    async (show: boolean) => {
      if (!workspaceSlug) return;

      // When toggling off, set to 0; when toggling on, use current count or default
      const newLimit = show ? preferences.limitedProjectsCount || DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount : 0;

      await updateProjectNavigationPreferences(workspaceSlug, {
        navigation_project_limit: newLimit,
      });
    },
    [workspaceSlug, updateProjectNavigationPreferences, preferences.limitedProjectsCount]
  );

  // Update limited projects count
  const updateLimitedProjectsCount = useCallback(
    async (count: number) => {
      if (!workspaceSlug) return;

      await updateProjectNavigationPreferences(workspaceSlug, {
        navigation_project_limit: count,
      });
    },
    [workspaceSlug, updateProjectNavigationPreferences]
  );

  return {
    preferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  };
};
