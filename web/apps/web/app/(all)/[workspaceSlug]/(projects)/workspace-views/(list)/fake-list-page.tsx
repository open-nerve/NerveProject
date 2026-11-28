/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The workspace's views list page as its tests open it (M7/P6b spec 2.10): a tab of workspace ws whose caller has a
// role there (fake-view-page.tsx), with the workspaces given besides, two views of ws's own as nerve gives them (First,
// the caller's; Second, Ann's), and the page's layout and page at the app's routes, SWR's cache the test's own; what a
// person reads of the page (its rows, its search, its loader, "Add view", a row's menus) and does on it.
import { act, type ComponentType } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { vi } from "vitest";
import type { View, Workspace, WorkspaceRole } from "@nerve/api-client";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { offered, settle, Where } from "@/components/common/fake-quick-actions";
import { workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import Layout from "./layout";
import Page from "./page";

/** The default views' names, as the fake translation gives them (their keys), and their keys, in their order. */
export const DEFAULTS = DEFAULT_GLOBAL_VIEWS_LIST.map((view) => view.i18n_label);
export const DEFAULT_KEYS = DEFAULT_GLOBAL_VIEWS_LIST.map((view) => view.key);

type RouteProps = { params: Readonly<Record<string, string | undefined>> };
function RouteOf({ Component }: { Component: unknown }) {
  const Of = Component as ComponentType<RouteProps>;
  return <Of params={useParams()} />;
}

/** A tab of ws whose caller has role there, and the workspaces others too; open renders the list at path. */
export async function listTab(role: WorkspaceRole, others: Workspace[] = []) {
  const tab = await workspaceViewPageTab(role);
  if (others.length > 0) await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, ...others]);
  tab.nerve.calls.length = 0;
  return {
    ...tab,
    first: tab.view("First"),
    second: tab.view("Second", { owned_by_id: "u-ann" }),
    open: async (path = "/ws/workspace-views") => {
      await tab.render(
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter initialEntries={[path]}>
            <Routes>
              <Route path=":workspaceSlug/workspace-views" element={<RouteOf Component={Layout} />}>
                <Route index element={<RouteOf Component={Page} />} />
              </Route>
              <Route path=":workspaceSlug/workspace-views/:globalViewId" element={<p>a view&apos;s page</p>} />
            </Routes>
            <Where />
          </MemoryRouter>
        </SWRConfig>
      );
      await settle();
    },
  };
}

/** nerve answers the last read of the views of the workspace of slug (ws unless given) with views (First, Second). */
export async function answerViews(tab: Awaited<ReturnType<typeof listTab>>, views?: View[], slug = "ws") {
  const reads = () => tab.nerve.to(`/api/v0/workspaces/${slug}/views`).filter((call) => call.method === "GET");
  await until(() => reads().length > 0, `the views of ${slug}`);
  await act(async () =>
    reads()
      .at(-1)!
      .answer(json(200, { data: views ?? [tab.first, tab.second] }))
  );
  await settle();
}

/** The links of the list's rows, in its order: each one's name and address. */
const links = () => [...document.querySelectorAll<HTMLAnchorElement>("a[href*='/workspace-views/']")];
export const rows = () => links().map((link) => [link.textContent, link.getAttribute("href")]);

/** Whether the list is waiting for the workspace's views (the loader's marker). */
export const loading = () => document.querySelector('[role="status"]') !== null;

/** The header's "Add view", if the page offers it. */
export const addView = () =>
  [...document.querySelectorAll<HTMLButtonElement>("header button")].find(
    (button) => button.textContent?.trim() === "workspace_views.add_view"
  );

/** The page's search, by its name. */
export const searchBox = () => document.querySelector<HTMLInputElement>('input[aria-label="common.search.label"]');

/** Types query into the page's search, as a person does. */
export async function search(query: string) {
  const input = searchBox();
  await act(async () => {
    if (!input) return;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, query);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

/** The row of the view named name: the nearest box around its link that holds a menu's button; none without one. */
export function rowOf(name: string): HTMLElement | null {
  let at = links().find((link) => link.textContent === name)?.parentElement ?? null;
  while (at && at !== document.body && !at.querySelector('button[aria-haspopup="menu"]')) at = at.parentElement;
  return at === document.body ? null : at;
}

/** The buttons of the row's menus. */
export const menuButtons = (row: HTMLElement | null) => [
  ...(row?.querySelectorAll<HTMLButtonElement>('button[aria-haspopup="menu"]') ?? []),
];

/** Opens the row's menu by its button: what it offers; then closes it. */
export async function menuOf(row: HTMLElement | null) {
  const [button] = menuButtons(row);
  await act(async () => {
    button?.focus();
    button?.click();
  });
  await settle();
  const menu = document.getElementById(button?.getAttribute("aria-controls") ?? "");
  const items = offered([...(menu?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])]);
  await act(async () => button?.click());
  await settle();
  return items;
}

/** The row's right-click menus: each one, whether it is out of reach (closed) and what it offers. */
export const rightClickMenus = (row: HTMLElement | null) =>
  [...(row?.querySelectorAll<HTMLElement>('[data-context-menu="true"]') ?? [])].map((panel) => ({
    closed: panel.closest("[inert]") !== null,
    items: offered([...panel.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')]),
  }));

/** Right-clicks the row, as a person does. */
export async function rightClick(row: HTMLElement | null) {
  await act(async () => row?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await act(async () => vi.advanceTimersByTimeAsync(100));
}
