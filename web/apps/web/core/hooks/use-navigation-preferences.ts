/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import type { TProjectNavigationPreferences } from "@nerve/types";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import type { TProjectNavigationChange } from "./navigation-preferences";
import { navigationOf, preferencesChangeOf } from "./navigation-preferences";
import { useWorkspace } from "./store/use-workspace";

/**
 * The sidebar's project navigation in the address's workspace, as the caller's settings there say (M3 design 3.18,
 * 7.5), and its change: made in its turn to the settings nerve last answered (navigation-preferences.ts), and followed
 * only while the tab stays in the session it was sent in (M3 design 7.1): a refusal's reason is said in a toast, and
 * nothing once another tab has moved this one to another account.
 */
export function useProjectNavigationPreferences(): {
  preferences: TProjectNavigationPreferences;
  changeNavigation: (change: TProjectNavigationChange) => Promise<void>;
} {
  const { workspaceSlug = "" } = useParams();
  const toastRefusal = useRefusalToast();
  const {
    preferences: { getPreferences, updatePreferences },
  } = useWorkspace();
  return {
    preferences: navigationOf(getPreferences(workspaceSlug)),
    changeNavigation: (change) =>
      followInSession(() => updatePreferences(workspaceSlug, preferencesChangeOf(change)), {
        failed: toastRefusal,
      }),
  };
}
