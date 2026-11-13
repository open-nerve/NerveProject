// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a module's page's work items offer (P6b spec 2.7, R9, R11, R15): once the module is archived (whatever status
// nerve gives it), each layout is read-only, for its project's members too: no quick add, group "+", inline edit or
// drag, and a row's quick actions are only its opening in a new tab and the copy of its link. A module completed or
// cancelled is no read-only one: each layout offers its project's members what the project page does. A guest sees the
// work items without any write. With no work items, the page offers a new work item of the module to those who may
// create one while the module takes them, the clearing of its filters while any is applied (to a guest too), and, once
// the module is archived, only why there are none. The layouts, rows, menus, stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { M1, buttonOf, layouts, modulePreferencesOf, planningPageTab, type Layout } from "./fake-planning-page";

type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  drags: new Map<Element, Drag>(),
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
// jsdom has no viewport geometry or drag transport; rows, cards, days, menus, dropdowns and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  draggable: (options: Drag) => {
    bound.drags.set(options.element, options);
    return () => bound.drags.delete(options.element);
  },
}));

let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the module's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.MODULE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  bound.drags.clear();
});
afterEach(async () => page.close());

// nerve archives only completed or cancelled modules: being archived alone makes the list read-only, its status none
const archived: Partial<Module> = { status: "completed", archived_at: "2026-12-01T09:00:00Z" };
const ARCHIVED_EMPTY = "project_module.empty_state.archived.title";
const CREATE = "project_empty_state.module_work_items.cta_primary";
const CLEAR = "project_issues.empty_state.issues_empty_filter.secondary_button.text";
const text = () => document.body.textContent ?? "";
const priorities = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((node) =>
    node.textContent?.startsWith("common.priority: ")
  );
/** What the shown layout offers: creation, inline edits of the row, dragging it, and its quick actions by name. */
function offered() {
  const buttons = [...document.querySelectorAll<HTMLButtonElement>("button")];
  return {
    quickAdd: buttons.some((node) => ["issue.new", "issue.add.label"].includes(node.textContent?.trim() ?? "")),
    groupCreate: !!document.querySelector('button[aria-label^="create_work_item: "]'),
    inlineEdit: priorities().some((node) => !node.disabled),
    drag: [...bound.drags.values()].some((drag) => drag.canDrag?.({} as never) ?? true),
    quickActions: [
      ...new Set(
        [...document.querySelectorAll('[data-context-menu="true"] [role="menuitem"]')].map((node) => node.textContent)
      ),
    ],
  };
}
const none = {
  quickAdd: false,
  groupCreate: false,
  inlineEdit: false,
  drag: false,
  quickActions: ["common.actions.open_in_new_tab", "common.actions.copy_link"],
};
/** Each layout's writes the project page offers a member: its group "+", inline edits and drag. */
const writes: Record<Layout, { groupCreate: boolean; inlineEdit: boolean; drag: boolean }> = {
  list: { groupCreate: true, inlineEdit: true, drag: true },
  kanban: { groupCreate: true, inlineEdit: true, drag: true },
  spreadsheet: { groupCreate: false, inlineEdit: true, drag: false },
  calendar: { groupCreate: false, inlineEdit: false, drag: true },
};
/** Shows the module's page in layout, as fields say, its one work item or none answered. */
async function shown(fields: Parameters<typeof planningPageTab>[1], layout: Layout, rows = true, filters = {}) {
  await opened(fields);
  await page.show(modulePreferencesOf(layouts[layout].display, filters), page.listOf(layout, rows));
  if (rows) await until(() => document.querySelectorAll('[id^="issue-first"]').length > 0, "the row");
}

it.each(Object.keys(layouts) as Layout[])("the %s of an archived module is read-only for a member", async (layout) => {
  await shown({ module: archived }, layout);
  expect(offered()).toEqual(none);
});

it.each(
  (Object.keys(layouts) as Layout[]).flatMap((layout) =>
    (["completed", "cancelled"] as const).map((status) => ({ layout, status }))
  )
)("the $layout of a module $status offers a member each write the project page does", async ({ layout, status }) => {
  await shown({ module: { status } }, layout);
  const { quickActions, ...rest } = offered();
  expect(rest).toEqual({ quickAdd: true, ...writes[layout] });
  expect(quickActions).toEqual(
    expect.arrayContaining(["common.actions.edit", "common.actions.make_a_copy", "common.actions.delete"])
  );
});

it("shows a guest the module's work items without a write", async () => {
  await shown({ role: 5 }, "list");
  expect(offered()).toEqual(none);
});

it("offers a member a new work item of the module when it has none", async () => {
  await shown({}, "list", false);
  expect(text()).toContain("project_empty_state.module_work_items.title");
  const create = buttonOf(CREATE);
  expect(create?.disabled).toBe(false);
  await act(async () => create?.click());
  expect(page.app.commandPalette.isCreateIssueModalOpen).toBe(true);
  expect(page.app.commandPalette.createIssueStoreType).toBe(EIssuesStoreType.MODULE);
});

it("offers a guest no new work item when the module has none", async () => {
  await shown({ role: 5 }, "list", false);
  expect(text()).toContain("project_empty_state.module_work_items.title");
  expect(buttonOf(CREATE)).toBeUndefined();
});

it.each([
  { who: "a member", role: 15 as const },
  { who: "a guest", role: 5 as const },
])("offers $who the clearing of the page's filters once they leave no work item", async ({ role }) => {
  await shown({ role }, "list", false, { priority: ["high"] });
  expect(text()).toContain("common_empty_state.search.title");
  expect(buttonOf(CREATE)).toBeUndefined();
  await act(async () => buttonOf(CLEAR)?.click());
  await until(() => page.preferences().length === 2, "the change of the page's filters");
  expect(page.preferences()[1]).toMatchObject({ method: "PATCH", body: { filters: {} } });
});

it.each([{ filters: {} }, { filters: { priority: ["high"] } }])(
  "only says why an archived module has no work items, with filters $filters",
  async ({ filters }) => {
    await shown({ module: archived }, "list", false, filters);
    expect(text()).toContain(ARCHIVED_EMPTY);
    expect(text()).not.toContain("project_empty_state.module_work_items.title");
    expect(buttonOf(CREATE)).toBeUndefined();
    expect(buttonOf(CLEAR)).toBeUndefined();
  }
);
