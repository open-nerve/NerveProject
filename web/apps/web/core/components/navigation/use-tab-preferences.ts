/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import type { ProjectTab } from "@nerve/api-client";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { DEFAULT_TAB_KEY, hideTab, showTab, toggleDefaultTab } from "./tab-navigation-utils";
import type { TTabPreferences } from "./tab-navigation-utils";

export type TTabPreferencesHook = {
  tabPreferences: TTabPreferences;
  handleToggleDefaultTab: (tabKey: ProjectTab) => void;
  handleHideTab: (tabKey: ProjectTab) => void;
  handleShowTab: (tabKey: ProjectTab) => void;
};

/**
 * The caller's tab bar in the project's header (ProjectPreferences.navigation): the tab the project opens on and the
 * tabs under "more", shown as nerve's default (work items, none hidden) until the project wrapper has fetched his. A
 * change shows once nerve has answered it; one nerve refuses leaves the tab bar as it was. Each change is made, in its
 * turn, to the tab bar nerve last answered (the store), so a change asked for before the one before it is answered
 * keeps that one; one asked for before his tab bar is fetched fails without being sent, as nerve would replace his
 * tab bar with it. A change that fails shows an error toast.
 *
 * @param projectId - The project ID
 * @returns Tab preferences state and handlers
 */
export const useTabPreferences = (projectId: string): TTabPreferencesHook => {
  const { getNavigation, updateNavigation } = useProjectPreferences();
  const navigation = getNavigation(projectId);

  const tabPreferences: TTabPreferences = useMemo(
    () => ({
      defaultTab: navigation?.default_tab ?? DEFAULT_TAB_KEY,
      hiddenTabs: navigation?.hide_in_more_menu ?? [],
    }),
    [navigation]
  );

  /**
   * Toggle default tab setting
   * If tab is already default, resets to work_items; otherwise sets as default
   */
  const handleToggleDefaultTab = (tabKey: ProjectTab) => {
    updateNavigation(projectId, toggleDefaultTab(tabKey))
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Default tab updated successfully.",
        });
        return;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Failed to update default tab. Please try again later.",
        });
      });
  };

  /**
   * Hide a tab (moves to overflow menu with "Show" option)
   */
  const handleHideTab = (tabKey: ProjectTab) => {
    updateNavigation(projectId, hideTab(tabKey)).catch((error: unknown) => {
      console.error("Error hiding tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to hide tab. Please try again later.",
      });
    });
  };

  /**
   * Show a previously hidden tab (returns to visible pool)
   */
  const handleShowTab = (tabKey: ProjectTab) => {
    updateNavigation(projectId, showTab(tabKey)).catch((error: unknown) => {
      console.error("Error showing tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again later.",
      });
    });
  };

  return {
    tabPreferences,
    handleToggleDefaultTab,
    handleHideTab,
    handleShowTab,
  };
};
