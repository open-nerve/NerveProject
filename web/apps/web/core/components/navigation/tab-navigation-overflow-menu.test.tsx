/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { buttonsIn, emptyShown, shown } from "@/lib/fake-controls";
import { TabNavigationOverflowMenu } from "./tab-navigation-overflow-menu";
import type { TNavigationItem } from "./tab-navigation-root";
import { DEFAULT_NAVIGATION } from "./tab-navigation-utils";
import type { TTabChanges } from "./use-tab-preferences";

// What the "more" menu of a project's header offers for each tab it holds (M3 design 3.18, 7.1): no change until nerve
// has given the caller's tab bar (a change made to nerve's default would replace his whole), the tabs there being the
// ones the header has no room for; then, for a tab he moved there, showing it again, and for each, making it the tab
// the project opens on, or no longer. The menu renders on the server with a stand-in for propel's menu
// (fake-controls.ts), which renders its items as the open menu does and keeps each; the test reads each item's buttons
// and clicks them.

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

/** A click on a button of the menu's. */
const click = { stopPropagation: () => undefined, preventDefault: () => undefined };

/** The menu holding cycles and modules, with the tab bar and the changes given: the buttons of each of its items. */
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
  return shown.menuItems.map(({ children }) => buttonsIn(children));
}

beforeEach(() => {
  changes.toggleDefault.mockClear();
  changes.hide.mockClear();
  changes.show.mockClear();
});

describe("the more menu of a project's header", () => {
  it("offers no change before nerve has given the caller's tab bar", () => {
    const items = offered(DEFAULT_NAVIGATION, undefined);
    expect(items.map((buttons) => buttons.map(({ title }) => title))).toEqual([[], []]);
  });

  it("then shows again a tab he moved there, and makes each the tab the project opens on, or no longer", () => {
    const items = offered({ default_tab: "cycles", hide_in_more_menu: ["modules"] }, changes);
    expect(items.map((buttons) => buttons.map(({ title }) => title))).toEqual([
      ["Clear default"],
      ["Show", "Set as default"],
    ]);
    for (const button of items.flat()) button.onClick?.(click);
    expect([changes.toggleDefault.mock.calls, changes.show.mock.calls, changes.hide.mock.calls]).toEqual([
      [["cycles"], ["modules"]],
      [["modules"]],
      [],
    ]);
  });
});
