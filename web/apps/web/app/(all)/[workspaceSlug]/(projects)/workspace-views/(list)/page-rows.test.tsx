// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The menus of the workspace's views list's rows (M7/P6b spec 2.7, 2.10, 2.11, R12): each row has one menu, by its
// button, named by the view, and by a right-click on the row, out of reach until opened, which offers what the button's
// does. A default row's (default-view-menu.tsx) opens its page in a new tab and copies its link, nothing else, for
// anyone of the workspace; a saved row's is its view's (quick-actions.tsx, by the workspace as its container): what
// nerve lets the caller do with the view, by his role in the workspace. The stores, the layout, SWR and HTTP are real
// (fake-list-page.tsx), at the app's routes; the app's header a frame that shows the header.
import { type ReactNode } from "react";
import { afterEach, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import {
  answerViews,
  DEFAULTS,
  listTab,
  menuButtons,
  menuOf,
  rightClick,
  rightClickMenus,
  rowOf,
} from "./fake-list-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

let tab: Awaited<ReturnType<typeof listTab>>;
afterEach(async () => tab.close());

/** A tab of ws whose caller has role there, at ws's views list, its own views answered when answered says so. */
async function opened(role: WorkspaceRole, answered: boolean) {
  tab = await listTab(role);
  bound.app = tab.app;
  await tab.open();
  if (answered) await answerViews(tab);
}
const DEFAULT_MENU = ["open_in_new_tab", "copy_link"];

it.each([
  ["its admin", 20],
  ["its member", 15],
  ["its guest", 5],
] as const)(
  "gives each default row, to %s (role %i), one menu named by the view, by its button and by a right-click: open and copy",
  async (_, role) => {
    await opened(role, false);
    for (const name of DEFAULTS) {
      const row = rowOf(name);
      expect(
        menuButtons(row).map((button) => button.getAttribute("aria-label")),
        name
      ).toEqual([`common.item_options ${name}`]);
      expect(rightClickMenus(row), name).toEqual([{ closed: true, items: DEFAULT_MENU }]);
    }
    const assigned = rowOf("default_global_view.assigned");
    expect(await menuOf(assigned)).toEqual(DEFAULT_MENU);
    await rightClick(assigned);
    expect(rightClickMenus(assigned)).toEqual([{ closed: false, items: DEFAULT_MENU }]);
  }
);

it.each([
  ["its member, of a view of his", 15, "First", ["edit", "open_in_new_tab", "copy_link", "delete"]],
  ["its member, of another's view", 15, "Second", ["open_in_new_tab", "copy_link"]],
  ["its admin, of another's view", 20, "Second", ["open_in_new_tab", "copy_link", "delete"]],
] as const)(
  "offers %s, by the row's menu named by the view and by a right-click, what nerve lets him do with it",
  async (_, role, name, items) => {
    await opened(role, true);
    const row = rowOf(name);
    expect(menuButtons(row).map((button) => button.getAttribute("aria-label"))).toEqual([
      `common.item_options ${name}`,
    ]);
    expect(await menuOf(row)).toEqual(items);
    await rightClick(row);
    expect(rightClickMenus(row)).toEqual([{ closed: false, items }]);
  }
);
