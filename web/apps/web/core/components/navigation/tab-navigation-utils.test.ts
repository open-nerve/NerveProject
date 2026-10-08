/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import type { NavigationChange } from "@/store/project/preferences.store";
import { hideTab, showTab, toggleDefaultTab } from "./tab-navigation-utils";

// The changes the header's controls make of the caller's tab bar (the store makes each to the tab bar nerve last
// answered): each row is a tab bar nerve answered, the change, and what the change makes of it, whole.

describe("the tab bar's changes", () => {
  it.each<[string, ProjectNavigation, NavigationChange, ProjectNavigation]>([
    [
      "toggling another tab makes it the one the project opens on",
      { default_tab: "work_items", hide_in_more_menu: ["views"] },
      toggleDefaultTab("cycles"),
      { default_tab: "cycles", hide_in_more_menu: ["views"] },
    ],
    [
      "toggling the tab it opens on opens it on its work items again",
      { default_tab: "modules", hide_in_more_menu: ["views"] },
      toggleDefaultTab("modules"),
      { default_tab: "work_items", hide_in_more_menu: ["views"] },
    ],
    [
      "hiding a tab puts it under more, last",
      { default_tab: "modules", hide_in_more_menu: ["views"] },
      hideTab("cycles"),
      { default_tab: "modules", hide_in_more_menu: ["views", "cycles"] },
    ],
    [
      "hiding a tab under more already lists it once",
      { default_tab: "modules", hide_in_more_menu: ["cycles", "views"] },
      hideTab("views"),
      { default_tab: "modules", hide_in_more_menu: ["cycles", "views"] },
    ],
    [
      "showing a tab takes it alone from under more",
      { default_tab: "modules", hide_in_more_menu: ["views", "cycles", "intake"] },
      showTab("cycles"),
      { default_tab: "modules", hide_in_more_menu: ["views", "intake"] },
    ],
  ])("%s", (_, held, change, made) => {
    expect(change(held)).toEqual(made);
  });
});
