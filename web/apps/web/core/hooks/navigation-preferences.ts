/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NavigationControlPreference, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import { DEFAULT_PROJECT_PREFERENCES } from "@nerve/types";

/** One change of the sidebar's project navigation: its mode, whether it limits the projects, or to how many. */
export type TProjectNavigationChange =
  | { navigationMode: NavigationControlPreference }
  | { showLimitedProjects: boolean }
  | { limitedProjectsCount: number };

/**
 * The sidebar's project navigation as the caller's settings in a workspace say (M3 design 3.18): a limit of 0 shows
 * every project, and the count is then DEFAULT_PROJECT_PREFERENCES's, which turning the limit on starts from. Until
 * the settings load it is DEFAULT_PROJECT_PREFERENCES, which shows every project, where nerve's defaults show 10.
 */
export function navigationOf(preferences: WorkspacePreferences | undefined): TProjectNavigationPreferences {
  if (!preferences) return DEFAULT_PROJECT_PREFERENCES;
  const limit = preferences.navigation_project_limit;
  return {
    navigationMode: preferences.navigation_control_preference,
    limitedProjectsCount: limit > 0 ? limit : DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount,
    showLimitedProjects: limit > 0,
  };
}

/**
 * The change of the caller's settings that a change of the sidebar's project navigation is, given what it shows:
 * turning the limit on limits it to the count shown, and turning it off is a limit of 0.
 */
export function preferencesChangeOf(
  change: TProjectNavigationChange,
  shown: TProjectNavigationPreferences
): WorkspacePreferencesUpdate {
  if ("navigationMode" in change) return { navigation_control_preference: change.navigationMode };
  if ("showLimitedProjects" in change) {
    return { navigation_project_limit: change.showLimitedProjects ? shown.limitedProjectsCount : 0 };
  }
  return { navigation_project_limit: change.limitedProjectsCount };
}
