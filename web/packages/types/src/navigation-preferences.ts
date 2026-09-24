/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type TProjectNavigationMode = "ACCORDION" | "TABBED";

export interface TProjectDisplaySettings {
  navigationMode: TProjectNavigationMode;
  showLimitedProjects: boolean;
  limitedProjectsCount: number;
}

export interface TProjectNavigationPreferences {
  navigationMode: TProjectNavigationMode;
  showLimitedProjects: boolean;
  limitedProjectsCount: number;
}

export const DEFAULT_PROJECT_PREFERENCES: TProjectNavigationPreferences = {
  navigationMode: "ACCORDION",
  showLimitedProjects: false,
  limitedProjectsCount: 10,
};
