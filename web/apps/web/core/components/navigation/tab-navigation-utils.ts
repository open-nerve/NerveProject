/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import type { NavigationChange } from "@/store/project/preferences.store";

export const DEFAULT_TAB_KEY: ProjectTab = "work_items";
/** nerve's tab bar until the caller changes his: the project opens on its work items, and no tab is under "more". */
export const DEFAULT_NAVIGATION: ProjectNavigation = { default_tab: DEFAULT_TAB_KEY, hide_in_more_menu: [] };

// The changes of the caller's tab bar that the header's controls make, each to the tab bar nerve last answered
// (ProjectPreferencesStore.updateNavigation)

/** The project opens on tab; on its work items again when it opens on tab already. */
export const toggleDefaultTab =
  (tab: ProjectTab): NavigationChange =>
  (held) => ({ ...held, default_tab: tab === held.default_tab ? DEFAULT_TAB_KEY : tab });

/** tab goes under "more", last; a tab there already is not listed twice. */
export const hideTab =
  (tab: ProjectTab): NavigationChange =>
  (held) => ({ ...held, hide_in_more_menu: [...held.hide_in_more_menu.filter((hidden) => hidden !== tab), tab] });

/** tab leaves "more"; the other tabs there stay. */
export const showTab =
  (tab: ProjectTab): NavigationChange =>
  (held) => ({ ...held, hide_in_more_menu: held.hide_in_more_menu.filter((hidden) => hidden !== tab) });

/**
 * Map tab keys to their corresponding URLs
 * @param workspaceSlug - The workspace slug
 * @param projectId - The project ID
 * @param tabKey - The tab key to map
 * @returns Full URL path for the tab
 */
export const getTabUrl = (workspaceSlug: string, projectId: string, tabKey: string): string => {
  const baseUrl = `/${workspaceSlug}/projects/${projectId}`;
  const tabUrlMap: Record<string, string> = {
    work_items: `${baseUrl}/issues`,
    cycles: `${baseUrl}/cycles`,
    modules: `${baseUrl}/modules`,
    views: `${baseUrl}/views`,
    intake: `${baseUrl}/intake`,
  };
  return tabUrlMap[tabKey] || `${baseUrl}/issues`; // fallback to issues
};
