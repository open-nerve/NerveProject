/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import type { TNavigationItem } from "./tab-navigation-root";
import { DEFAULT_NAVIGATION } from "./tab-navigation-utils";
import { TabNavigationVisibleItem } from "./tab-navigation-visible-item";
import type { TTabChanges } from "./use-tab-preferences";

// What the menu of a tab of a project's header offers, in the page's language (M3 design 3.18): making the tab the one
// the project opens on, or no longer, and moving it to the "more" menu. The tab renders on the server with a stand-in
// for propel's context menu, which renders its items as the open menu does, and the language says each key
// (fake-i18n.ts).

vi.mock("@nerve/propel/context-menu", () => ({
  ContextMenu: Object.assign(Shown, { Trigger: Shown, Portal: Shown, Content: Shown, Item: Shown }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each part of the menu, shown with what it holds. */
function Shown({ children }: { children?: ReactNode }) {
  return <>{children}</>;
}

const cycles: TNavigationItem = {
  name: "cycles",
  href: "/acme/projects/p-web/cycles",
  icon: () => null,
  access: [],
  shouldRender: true,
  sortOrder: 2,
  i18n_key: "sidebar.cycles",
  key: "cycles",
};
const changes = { toggleDefault: () => undefined, hide: () => undefined, show: () => undefined } satisfies TTabChanges;

/** The menu of the cycles' tab, with the tab bar given. */
function menuOf(navigation: ProjectNavigation): string {
  return renderToStaticMarkup(
    <MemoryRouter>
      <TabNavigationVisibleItem item={cycles} isActive={false} navigation={navigation} changes={changes} />
    </MemoryRouter>
  );
}

describe("the menu of a tab of a project's header", () => {
  it("offers making the tab the one the project opens on, or no longer, and moving it to the more menu", () => {
    const menu = menuOf(DEFAULT_NAVIGATION);
    expect([menu.includes("tab_navigation.set_as_default"), menu.includes("tab_navigation.hide_in_more_menu")]).toEqual(
      [true, true]
    );
    expect(menuOf({ ...DEFAULT_NAVIGATION, default_tab: "cycles" })).toContain("tab_navigation.clear_default");
  });
});
