// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row's edit and copy dialogs outlive the row (an M4 defect found in M6 P5b T13). The dialog reads its work item
// with its resources; the read of its sub-issues checks the work item itself against nerve (M4 P12 D22), and while
// that check is out the list takes the row off the page, then reads its first page again. The layout's command host
// holds the dialog, so it stays open with what it was opened with while the row goes and comes back, on the project's
// page and in each layout of a cycle's page. The pages, layouts, rows, menus, the dialog, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  choose,
  dialogTitle,
  outlivesTheRow,
  planningPageTab,
  cyclePreferencesOf,
  layouts,
  rowShown,
  type Layout,
} from "./fake-planning-page";
import { ProjectLayoutRoot } from "./project-layout-root";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
}));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; rows, cards, days and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => page.close());

/** Shows the cycle's page in layout, its one work item answered. */
async function cyclePage(layout: Layout) {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
  return layouts[layout].page();
}
/** Shows the project's page in its list grouped by state, of the same one work item. */
async function projectPage() {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const preferences = () => page.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the project's preferences");
  const display = { ...preferencesOf().display_filters, ...layouts.list.display };
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf({ display_filters: display }))));
  await until(() => page.lists().length === 1, "the project's list");
  const list = layouts.list.page();
  await act(async () => page.lists()[0]!.answer(json(200, list)));
  return list;
}

it.each(Object.keys(layouts) as Layout[])(
  "keeps a row's edit dialog open with its work item in a cycle's %s layout while the row is checked",
  async (layout) => {
    const list = await cyclePage(layout);
    await outlivesTheRow("common.actions.edit", "first", list);
  }
);

it("keeps a row's copy dialog open with the copy on a cycle's page while the row is checked", async () => {
  const list = await cyclePage("list");
  await outlivesTheRow("common.actions.make_a_copy", "first (copy)", list);
});

it.each([
  ["edit", "common.actions.edit", "first"],
  ["copy", "common.actions.make_a_copy", "first (copy)"],
])("keeps a row's %s dialog open on the project's page while the row is checked", async (_, name, title) => {
  const list = await projectPage();
  await outlivesTheRow(name, title, list);
});

// The layout ends a row's dialog as it ends a row's confirmation: once the caller's permission in the project changes.
it("ends a row's edit dialog once the caller's role in the project changes", async () => {
  await cyclePage("list");
  await until(rowShown, "the row");
  await choose("common.actions.edit");
  expect(dialogTitle()).toBe("first");
  await act(async () =>
    page.app.projectRoot.project.confirmProject(page.project, (project) => ({ ...project, member_role: 20 }))
  );
  expect(dialogTitle()).toBeNull();
});
