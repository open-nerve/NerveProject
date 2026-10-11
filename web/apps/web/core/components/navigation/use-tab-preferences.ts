/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import type { NavigationChange } from "@/store/project/preferences.store";
import { DEFAULT_NAVIGATION, hideTab, showTab, toggleDefaultTab } from "./tab-navigation-utils";

/** The changes of the caller's tab bar that the header's controls make, each of the tab it is given. */
export type TTabChanges = {
  toggleDefault: (tabKey: ProjectTab) => void;
  hide: (tabKey: ProjectTab) => void;
  show: (tabKey: ProjectTab) => void;
};

export type TTabPreferencesHook = {
  navigation: ProjectNavigation;
  changes: TTabChanges | undefined;
};

/**
 * The caller's tab bar in the project's header (ProjectPreferences.navigation): the tab the project opens on and the
 * tabs under "more", shown as nerve's default (work items, none hidden) until the project wrapper has fetched his; and
 * its changes, none until then, so that the header offers none: the store makes a change to the tab bar nerve last
 * answered, and sends none before it has one (made to the default, it would replace his whole). A change shows once
 * nerve has answered it, and the page follows it in the session that sent it (M3 design 7.1): a new default says so; a
 * refusal says nerve's reason, the tab bar as it was.
 *
 * @param projectId - The project ID
 * @returns The caller's tab bar, nerve's default until it is fetched, and its changes once it is
 */
export const useTabPreferences = (projectId: string): TTabPreferencesHook => {
  const { getNavigation, updateNavigation } = useProjectPreferences();
  const toastRefusal = useRefusalToast();
  const fetched = getNavigation(projectId);

  const send = (change: NavigationChange, done?: () => void) =>
    void followInSession(() => updateNavigation(projectId, change), { done, failed: toastRefusal });

  return {
    navigation: fetched ?? DEFAULT_NAVIGATION,
    changes: fetched
      ? {
          toggleDefault: (tabKey) =>
            send(toggleDefaultTab(tabKey), () =>
              setToast({
                type: TOAST_TYPE.SUCCESS,
                title: "Success!",
                message: "Default tab updated successfully.",
              })
            ),
          hide: (tabKey) => send(hideTab(tabKey)),
          show: (tabKey) => send(showTab(tabKey)),
        }
      : undefined,
  };
};
