/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "next/navigation";
import type {
  TProjectNavigationPreferences,
  TProjectNavigationMode,
  TAppRailPreferences,
  TAppRailDisplayMode,
} from "@plane/types";
import { DEFAULT_PROJECT_PREFERENCES, DEFAULT_APP_RAIL_PREFERENCES } from "@plane/types";
import { useWorkspace } from "./store/use-workspace";
import useLocalStorage from "./use-local-storage";

const APP_RAIL_PREFERENCES_KEY = "app_rail_preferences";

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams();
  const { getProjectNavigationPreferences, updateProjectNavigationPreferences } = useWorkspace();

  // Get preferences from the store
  const storePreferences = getProjectNavigationPreferences(workspaceSlug?.toString() || "");

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

      await updateProjectNavigationPreferences(workspaceSlug.toString(), {
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

      await updateProjectNavigationPreferences(workspaceSlug.toString(), {
        navigation_project_limit: newLimit,
      });
    },
    [workspaceSlug, updateProjectNavigationPreferences, preferences.limitedProjectsCount]
  );

  // Update limited projects count
  const updateLimitedProjectsCount = useCallback(
    async (count: number) => {
      if (!workspaceSlug) return;

      await updateProjectNavigationPreferences(workspaceSlug.toString(), {
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

export const useAppRailPreferences = () => {
  const { storedValue, setValue } = useLocalStorage<TAppRailPreferences>(
    APP_RAIL_PREFERENCES_KEY,
    DEFAULT_APP_RAIL_PREFERENCES
  );

  const updateDisplayMode = useCallback(
    (mode: TAppRailDisplayMode) => {
      setValue({
        displayMode: mode,
      });
    },
    [setValue]
  );

  const toggleDisplayMode = useCallback(() => {
    const currentPreferences = storedValue || DEFAULT_APP_RAIL_PREFERENCES;
    const newMode = currentPreferences.displayMode === "icon_only" ? "icon_with_label" : "icon_only";
    updateDisplayMode(newMode);
  }, [storedValue, updateDisplayMode]);

  return {
    preferences: storedValue || DEFAULT_APP_RAIL_PREFERENCES,
    updateDisplayMode,
    toggleDisplayMode,
  };
};
