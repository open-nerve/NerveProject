/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import { DEFAULT_PROJECT_PREFERENCES } from "@nerve/types";
import type { TProjectNavigationChange } from "./navigation-preferences";
import { navigationOf, preferencesChangeOf } from "./navigation-preferences";

// The sidebar's project navigation and the caller's settings in a workspace (M3 design 3.18), both ways.

const views: { settings: string; preferences?: WorkspacePreferences; shown: TProjectNavigationPreferences }[] = [
  {
    settings: "a limit of 3, as an accordion",
    preferences: { navigation_control_preference: "ACCORDION", navigation_project_limit: 3 },
    shown: { navigationMode: "ACCORDION", limitedProjectsCount: 3, showLimitedProjects: true },
  },
  {
    settings: "a limit of 0, in tabs",
    preferences: { navigation_control_preference: "TABBED", navigation_project_limit: 0 },
    shown: { navigationMode: "TABBED", limitedProjectsCount: 10, showLimitedProjects: false },
  },
  { settings: "none yet", shown: DEFAULT_PROJECT_PREFERENCES },
];

/** The sidebar shows 7 projects, in tabs. */
const sevenTabbed: TProjectNavigationPreferences = {
  navigationMode: "TABBED",
  limitedProjectsCount: 7,
  showLimitedProjects: true,
};
const changes: { does: string; change: TProjectNavigationChange; sent: WorkspacePreferencesUpdate }[] = [
  {
    does: "the mode",
    change: { navigationMode: "ACCORDION" },
    sent: { navigation_control_preference: "ACCORDION" },
  },
  {
    does: "turning the limit off, a limit of 0",
    change: { showLimitedProjects: false },
    sent: { navigation_project_limit: 0 },
  },
  {
    does: "turning the limit on, the count shown",
    change: { showLimitedProjects: true },
    sent: { navigation_project_limit: 7 },
  },
  { does: "the count the caller set", change: { limitedProjectsCount: 4 }, sent: { navigation_project_limit: 4 } },
];

describe("navigationOf", () => {
  it.each(views)("shows what the caller's settings say: $settings", ({ preferences, shown }) => {
    expect(navigationOf(preferences)).toEqual(shown);
  });
});

describe("preferencesChangeOf", () => {
  it.each(changes)("sends only the setting a change changes: $does", ({ change, sent }) => {
    expect(preferencesChangeOf(change, sevenTabbed)).toEqual(sent);
  });
});
