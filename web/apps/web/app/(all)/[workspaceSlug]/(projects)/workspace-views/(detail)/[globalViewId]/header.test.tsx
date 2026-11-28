// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The header of a workspace's view's page (M7/P6b spec 2.7, 2.8, R6, R12): its crumb of the views links to their list;
// its switcher offers the four default views, and the workspace's own views to its admins and members, read once it
// opens (its guest's never read); "Add view" opens the page's dialog of a new view, for the workspace's admins and
// members; a default view's quick actions open its page in a new tab and copy its link, nothing else, a saved view's
// are its row's; its Display shows the page's own display, the spreadsheet's options without a cycle or the modules.
// The stores, the layouts, SWR and HTTP are real (fake-view-page.tsx), at the app's routes; the preview a marker.
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View, WorkspaceRole } from "@nerve/api-client";
import { dialog, settle, where } from "@/components/common/fake-quick-actions";
import { WorkspaceViewRoutes, workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import Layout from "../layout";
import Page from "./page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => <i data-peek="" /> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

const DEFAULTS = [
  "default_global_view.all_issues",
  "default_global_view.assigned",
  "default_global_view.created",
  "default_global_view.subscribed",
];
let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
let first: View, second: View;
beforeEach(() => {
  window.localStorage.clear();
});
afterEach(async () => page.close());

/** A tab of ws with the caller of role, at the page of key (a default view's key, or a view's id), its list answered. */
async function opened(role: WorkspaceRole, key = "all-issues") {
  page = await workspaceViewPageTab(role);
  bound.app = page.app;
  first = page.view("First");
  second = page.view("Second", { owned_by_id: "u-ann" });
  await page.render(
    <WorkspaceViewRoutes path={`/ws/workspace-views/${key}`} Layout={Layout} Page={Page} cache={new Map()} />
  );
  if (key === first.id) await page.answerRead(first);
  await page.answerList();
  await settle();
}
/** The button of the header whose text is name. */
const headerButton = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("header button")].find((b) => b.textContent?.trim() === name);
/** Opens the switcher of the header's crumb of the view, named name; its options. */
async function openSwitcher(name: string) {
  await act(async () => {
    headerButton(name)?.click();
    await vi.advanceTimersByTimeAsync(20);
  });
  return [...document.querySelectorAll<HTMLElement>('[role="option"]')];
}
/** Opens the header's quick actions: their items' titles. */
async function quickActions() {
  const button = document.querySelector<HTMLButtonElement>('header button[aria-haspopup="menu"]');
  await act(async () => button?.click());
  await settle();
  const menu = document.getElementById(button?.getAttribute("aria-controls") ?? "");
  return [...(menu?.querySelectorAll('[role="menuitem"] h5') ?? [])].map((item) => item.textContent);
}

it.each(["all-issues", "first"])("links its crumb of the views to the workspace's views list, on %s", async (key) => {
  await opened(15, key === "first" ? "v-ws-first" : key);
  const link = document.querySelector<HTMLAnchorElement>("header a[href='/ws/workspace-views']");
  expect(link?.textContent).toBe("views");
});

it("reads the workspace's own views only once its switcher opens, and goes to the view chosen there", async () => {
  await opened(15);
  expect(page.viewLists(), "the switcher closed").toEqual([]);
  await openSwitcher("default_global_view.all_issues");
  await until(() => page.viewLists().length === 1, "the workspace's views");
  await act(async () => page.viewLists()[0]!.answer(json(200, { data: [first, second] })));
  const options = [...document.querySelectorAll<HTMLElement>('[role="option"]')];
  expect(options.map((option) => option.textContent)).toEqual([...DEFAULTS, "First", "Second"]);
  await act(async () => {
    options[5]!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    options[5]!.click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(where.location?.pathname).toBe(`/ws/workspace-views/${second.id}`);
  expect(page.viewLists()).toHaveLength(1);
});

it("offers the workspace's guest the four default views alone in its switcher, reading none of its own", async () => {
  await opened(5);
  const options = await openSwitcher("default_global_view.all_issues");
  await settle();
  expect(options.map((option) => option.textContent)).toEqual(DEFAULTS);
  expect(page.viewLists()).toEqual([]);
});

it.each([
  ["its admin", 20, true],
  ["its member", 15, true],
  ["its guest", 5, false],
] as const)("offers %s (role %i) a new view of the workspace, by the page's dialog: %s", async (_, role, offered) => {
  await opened(role);
  const add = headerButton("workspace_views.add_view");
  expect(add !== undefined).toBe(offered);
  if (!add) return;
  await act(async () => add.click());
  await settle();
  expect(dialog()).not.toBeNull();
  expect(page.writes()).toEqual([]);
});

it("gives a default view's quick actions to open its page in a new tab and copy its link, a saved view's its row's", async () => {
  await opened(15);
  expect(await quickActions()).toEqual(["open_in_new_tab", "copy_link"]);
  await page.close();
  await opened(15, "v-ws-first");
  expect(await quickActions()).toEqual(["edit", "open_in_new_tab", "copy_link", "delete"]);
});

it("shows the page's own display in its Display: the spreadsheet's properties, the page's on, no cycle or modules", async () => {
  await opened(15);
  await act(async () => headerButton("common.display")?.click());
  await settle();
  const properties = [...document.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")];
  const names = properties.map((button) => button.textContent);
  expect(names).toContain("common.assignee");
  expect(names).not.toContain("common.cycle");
  expect(names).not.toContain("common.modules");
  // the page's display: every property shown, as a default view's display is before it changes
  const assignee = properties.find((button) => button.textContent === "common.assignee");
  expect(assignee?.getAttribute("aria-pressed")).toBe("true");
});
