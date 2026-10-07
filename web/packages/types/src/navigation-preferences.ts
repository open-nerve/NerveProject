/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { NavigationControlPreference } from "@nerve/api-client";

export interface TProjectNavigationPreferences {
  navigationMode: NavigationControlPreference;
  showLimitedProjects: boolean;
  limitedProjectsCount: number;
}

export const DEFAULT_PROJECT_PREFERENCES: TProjectNavigationPreferences = {
  navigationMode: "ACCORDION",
  showLimitedProjects: false,
  limitedProjectsCount: 10,
};
