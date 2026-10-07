/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { NavigationControlPreference } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import { DEFAULT_PROJECT_PREFERENCES } from "@nerve/types";
import { useWorkspace } from "./store/use-workspace";

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams();
  const {
    preferences: { getPreferences, updatePreferences },
  } = useWorkspace();

  // The caller's settings as nerve gave them: its defaults until he changes one (M3 design 3.18)
  const storePreferences = getPreferences(workspaceSlug || "");

  // Until they load, the sidebar uses the defaults; a limit of 0 shows every project
  const preferences: TProjectNavigationPreferences = useMemo(() => {
    if (!storePreferences) return DEFAULT_PROJECT_PREFERENCES;
    const limit = storePreferences.navigation_project_limit;
    return {
      navigationMode: storePreferences.navigation_control_preference,
      limitedProjectsCount: limit > 0 ? limit : DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount,
      showLimitedProjects: limit > 0,
    };
  }, [storePreferences]);

  // Update navigation mode
  const updateNavigationMode = useCallback(
    async (mode: NavigationControlPreference) => {
      if (!workspaceSlug) return;

      await updatePreferences(workspaceSlug, {
        navigation_control_preference: mode,
      });
    },
    [workspaceSlug, updatePreferences]
  );

  // Update show limited projects
  const updateShowLimitedProjects = useCallback(
    async (show: boolean) => {
      if (!workspaceSlug) return;

      // When toggling off, set to 0; when toggling on, use current count or default
      const newLimit = show ? preferences.limitedProjectsCount || DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount : 0;

      await updatePreferences(workspaceSlug, {
        navigation_project_limit: newLimit,
      });
    },
    [workspaceSlug, updatePreferences, preferences.limitedProjectsCount]
  );

  // Update limited projects count
  const updateLimitedProjectsCount = useCallback(
    async (count: number) => {
      if (!workspaceSlug) return;

      await updatePreferences(workspaceSlug, {
        navigation_project_limit: count,
      });
    },
    [workspaceSlug, updatePreferences]
  );

  return {
    preferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  };
};
