/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { isValidElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
import { TabNavigationOverflowMenu } from "./tab-navigation-overflow-menu";
import type { TNavigationItem } from "./tab-navigation-root";
import { DEFAULT_NAVIGATION } from "./tab-navigation-utils";
import type { TTabChanges } from "./use-tab-preferences";

// What the "more" menu of a project's header offers for each tab it holds (M3 design 3.18, 7.1): no change until nerve
// has given the caller's tab bar (a change made to nerve's default would replace his whole), the tabs there being the
// ones the header has no room for; then, for a tab he moved there, showing it again, and for each, making it the tab
// the project opens on, or no longer. Each is an item of the menu's, which the arrow keys reach and Enter does (M3
// closeout, C11-2): the tab a link to its page, each action titled by its tab. The menu renders on the server with a
// stand-in for propel's menu (fake-controls.ts), which renders its items as the open menu does and keeps each; the
// test reads each item and clicks it.

vi.mock("@nerve/propel/menu", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** A tab of Web's header, as the header lists it. */
const tabOf = (key: ProjectTab, sortOrder: number): TNavigationItem => ({
  name: key,
  href: `/acme/projects/p-web/${key}`,
  icon: () => null,
  access: [],
  shouldRender: true,
  sortOrder,
  i18n_key: `sidebar.${key}`,
  key,
});

const changes = {
  toggleDefault: vi.fn((_tabKey: ProjectTab) => undefined),
  hide: vi.fn((_tabKey: ProjectTab) => undefined),
  show: vi.fn((_tabKey: ProjectTab) => undefined),
} satisfies TTabChanges;

/**
 * The menu holding cycles and modules, with the tab bar and the changes given: each of its items, in order, as where
 * its link goes (none for an action) and what it is titled (none for a tab, which its text names).
 */
function offered(navigation: ProjectNavigation, tabChanges: TTabChanges | undefined) {
  emptyShown();
  renderToStaticMarkup(
    <MemoryRouter>
      <TabNavigationOverflowMenu
        overflowItems={[tabOf("cycles", 2), tabOf("modules", 3)]}
        isActive={() => false}
        navigation={navigation}
        changes={tabChanges}
      />
    </MemoryRouter>
  );
  return shown.menuItems;
}

/** Where item's link goes, if it renders as one; undefined for an item that is no link. */
const linkOf = ({ render }: (typeof shown.menuItems)[number]) =>
  isValidElement<{ to?: string }>(render) ? render.props.to : undefined;

const cycles = "/acme/projects/p-web/cycles";
const modules = "/acme/projects/p-web/modules";

beforeEach(() => {
  changes.toggleDefault.mockClear();
  changes.hide.mockClear();
  changes.show.mockClear();
});

describe("the more menu of a project's header", () => {
  it("offers each tab, a link to its page, and no change before nerve has given the caller's tab bar", () => {
    const items = offered(DEFAULT_NAVIGATION, undefined);
    expect(items.map((item) => [linkOf(item), item.title])).toEqual([
      [cycles, undefined],
      [modules, undefined],
    ]);
  });

  it("then shows again a tab he moved there, and makes each the tab the project opens on, or no longer", () => {
    const items = offered({ default_tab: "cycles", hide_in_more_menu: ["modules"] }, changes);
    expect(items.map((item) => [linkOf(item), item.title])).toEqual([
      [cycles, undefined],
      [undefined, "tab_navigation.clear_tab_default"],
      [modules, undefined],
      [undefined, "tab_navigation.show_tab"],
      [undefined, "tab_navigation.set_tab_as_default"],
    ]);
    for (const item of items) item.onClick?.();
    expect([changes.toggleDefault.mock.calls, changes.show.mock.calls, changes.hide.mock.calls]).toEqual([
      [["cycles"], ["modules"]],
      [["modules"]],
      [],
    ]);
  });
});
