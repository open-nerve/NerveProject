/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { NavigationControlPreference } from "@nerve/api-client";
import { navigationOf, preferencesChangeOf } from "./navigation-preferences";
import { useWorkspace } from "./store/use-workspace";

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams();
  const {
    preferences: { getPreferences, updatePreferences },
  } = useWorkspace();

  // The caller's settings as nerve gave them, in the sidebar's terms (navigation-preferences.ts)
  const storePreferences = getPreferences(workspaceSlug || "");
  const preferences = useMemo(() => navigationOf(storePreferences), [storePreferences]);

  // Update navigation mode
  const updateNavigationMode = useCallback(
    async (mode: NavigationControlPreference) => {
      if (!workspaceSlug) return;
      await updatePreferences(workspaceSlug, preferencesChangeOf({ navigationMode: mode }, preferences));
    },
    [workspaceSlug, updatePreferences, preferences]
  );

  // Update show limited projects
  const updateShowLimitedProjects = useCallback(
    async (show: boolean) => {
      if (!workspaceSlug) return;
      await updatePreferences(workspaceSlug, preferencesChangeOf({ showLimitedProjects: show }, preferences));
    },
    [workspaceSlug, updatePreferences, preferences]
  );

  // Update limited projects count
  const updateLimitedProjectsCount = useCallback(
    async (count: number) => {
      if (!workspaceSlug) return;
      await updatePreferences(workspaceSlug, preferencesChangeOf({ limitedProjectsCount: count }, preferences));
    },
    [workspaceSlug, updatePreferences, preferences]
  );

  return {
    preferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  };
};
