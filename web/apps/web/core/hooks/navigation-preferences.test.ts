/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import type { TProjectNavigationChange } from "./navigation-preferences";
import { countOf, navigationOf, preferencesChangeOf } from "./navigation-preferences";

// The sidebar's project navigation and the caller's settings in a workspace (M3 design 3.18), both ways, and the
// count the dialog's field gives.

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
  {
    settings: "none yet, taken to be nerve's defaults",
    shown: { navigationMode: "ACCORDION", limitedProjectsCount: 10, showLimitedProjects: true },
  },
];

/** The settings nerve last answered: a limit of 7, in tabs. */
const sevenTabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 7 };
/** The settings nerve last answered: every project, as an accordion. */
const all: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 };
const changes: {
  does: string;
  change: TProjectNavigationChange;
  held: WorkspacePreferences;
  sent: WorkspacePreferencesUpdate;
}[] = [
  {
    does: "the mode",
    change: { navigationMode: "ACCORDION" },
    held: sevenTabbed,
    sent: { navigation_control_preference: "ACCORDION" },
  },
  {
    does: "a turn of the limit, where the settings limit the projects: off, a limit of 0",
    change: { limitToggled: true },
    held: sevenTabbed,
    sent: { navigation_project_limit: 0 },
  },
  {
    does: "a turn of the limit, where the settings show every project: on, nerve's default count",
    change: { limitToggled: true },
    held: all,
    sent: { navigation_project_limit: 10 },
  },
  {
    does: "the count the caller set",
    change: { limitedProjectsCount: 4 },
    held: sevenTabbed,
    sent: { navigation_project_limit: 4 },
  },
];

describe("navigationOf", () => {
  it.each(views)("shows what the caller's settings say: $settings", ({ preferences, shown }) => {
    expect(navigationOf(preferences)).toEqual(shown);
  });
});

describe("preferencesChangeOf", () => {
  it.each(changes)(
    "sends only the setting a change changes, made to the settings nerve last answered: $does",
    ({ change, held, sent }) => {
      expect(preferencesChangeOf(change)(held)).toEqual(sent);
    }
  );
});

describe("countOf", () => {
  it.each([
    { draft: "3", count: 3 },
    { draft: "007", count: 7 },
    { draft: "0", count: undefined },
    { draft: "", count: undefined },
  ])("limits the sidebar to the number of a draft's digits, when 1 or more: '$draft'", ({ draft, count }) => {
    expect(countOf(draft)).toBe(count);
  });
});
