/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import { PROFILE_TABS } from "./profile";
import { GROUPED_PROFILE_SETTINGS, PROFILE_SETTINGS_TABS } from "./settings/profile";
import { WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS, WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS } from "./workspace";

// The sidebar is a fixed list (M1 design 3.3) and the profile page keeps only its three work-item tabs
// (M1 design 3.2). Both lists are the only navigation entry of the features they name, so a deletion
// that empties one of them has to fail here rather than silently remove the last way in.
describe("the workspace sidebar is a fixed list", () => {
  it("shows these items above the workspace group, in this order", () => {
    expect(WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS.map((item) => item.key)).toEqual([
      "home",
      "your_work",
      "drafts",
    ]);
  });

  it("shows these items inside the workspace group, in this order", () => {
    expect(WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS.map((item) => item.key)).toEqual([
      "projects",
      "views",
      "archives",
    ]);
  });

  it("gives every item a label, a link and at least one role", () => {
    for (const item of [
      ...WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,
      ...WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS,
    ]) {
      expect(item.labelTranslationKey).not.toBe("");
      expect(item.href.startsWith("/")).toBe(true);
      expect(item.access.length).toBeGreaterThan(0);
    }
  });
});

describe("the profile page", () => {
  it("keeps exactly the three work-item tabs", () => {
    expect(PROFILE_TABS.map((tab) => tab.key)).toEqual(["assigned", "created", "subscribed"]);
  });

  it("points each tab at its own route", () => {
    for (const tab of PROFILE_TABS) {
      expect(tab.route).toBe(tab.key);
      expect(tab.selected).toBe(`/${tab.key}/`);
    }
  });
});

// The profile settings are the only way to the password change (security) and to the personal access
// tokens (api-tokens); the email notification preferences left with all email sending (M1 design 2.2).
describe("the profile settings", () => {
  it("keeps exactly these tabs", () => {
    expect(PROFILE_SETTINGS_TABS).toEqual(["general", "security", "preferences", "api-tokens"]);
  });

  it("shows every tab in exactly one sidebar group", () => {
    const grouped = Object.values(GROUPED_PROFILE_SETTINGS).flatMap((items) => items.map((item) => item.key));
    expect(grouped.toSorted()).toEqual([...PROFILE_SETTINGS_TABS].toSorted());
  });
});
