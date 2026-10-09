/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NavigationControlPreference, WorkspacePreferences } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
// store
import type { PreferencesChange } from "@/store/workspace/preferences.store";

/** One change of the sidebar's project navigation: its mode, a turn of its limit, or the count it limits to. */
export type TProjectNavigationChange =
  | { navigationMode: NavigationControlPreference }
  | { limitToggled: true }
  | { limitedProjectsCount: number };

/** nerve's defaults: the caller's settings in a workspace until he changes one (M3 design 3.18). */
const NERVE_DEFAULTS: WorkspacePreferences = {
  navigation_control_preference: "ACCORDION",
  navigation_project_limit: 10,
};

/**
 * The sidebar's project navigation as the caller's settings in a workspace say (M3 design 3.18): a limit of 0 shows
 * every project, and the count is then nerve's default one, which turning the limit on starts from. Until the
 * settings arrive they are taken to be nerve's defaults, which are the settings of one who has changed none: his
 * sidebar does not change as they arrive.
 */
export function navigationOf(preferences: WorkspacePreferences = NERVE_DEFAULTS): TProjectNavigationPreferences {
  const limit = preferences.navigation_project_limit;
  return {
    navigationMode: preferences.navigation_control_preference,
    limitedProjectsCount: limit > 0 ? limit : NERVE_DEFAULTS.navigation_project_limit,
    showLimitedProjects: limit > 0,
  };
}

/**
 * The change of the caller's settings that a change of the sidebar's project navigation is, made in its turn to the
 * settings nerve last answered (v0 design 7.7), not to those the sidebar shows as it is asked for. A turn of the limit
 * is decided by those settings too: where they limit the projects it turns the limit off, a limit of 0; where they do
 * not, it turns it on, to the count they give (nerve's default one). Two quick turns end where they began.
 */
export function preferencesChangeOf(change: TProjectNavigationChange): PreferencesChange {
  return (held) => {
    if ("navigationMode" in change) return { navigation_control_preference: change.navigationMode };
    if ("limitToggled" in change) {
      const shown = navigationOf(held);
      return { navigation_project_limit: shown.showLimitedProjects ? 0 : shown.limitedProjectsCount };
    }
    return { navigation_project_limit: change.limitedProjectsCount };
  };
}

/** The count a draft of the dialog's count field limits the sidebar to: the number its digits make, when 1 or more. */
export function countOf(draft: string): number | undefined {
  const count = Number.parseInt(draft, 10);
  return count >= 1 ? count : undefined;
}
