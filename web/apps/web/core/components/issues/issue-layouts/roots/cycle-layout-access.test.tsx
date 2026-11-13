// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a cycle's page's work items offer (P5b spec 2.6, R5): once the cycle has ended or is archived (whatever status
// nerve gives it then), each layout is read-only, for its project's members too: no quick add, group "+", inline edit
// or drag, and the rows' quick actions are read-only. Else each layout offers them to those who may write in the
// project, and a guest sees the work items without any. With no work items, the page offers a new work item to those
// who may create one while the cycle takes them, the clearing of its filters while any is applied, and, once the cycle
// has ended or is archived, only why there are none.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import type { IQuickActionProps } from "../list/list-view-types";
import { C1, buttonOf, planningPageTab, cyclePreferencesOf, layouts, type Layout } from "./fake-planning-page";

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
// jsdom has no viewport geometry or drag transport; rows, cards, days, dropdowns and HTTP are real.
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
// A row's quick actions say whether they were given read-only.
vi.mock("../quick-action-dropdowns", async (original) => ({
  ...(await original<typeof import("../quick-action-dropdowns")>()),
  CycleIssueQuickActions: ({ issue, readOnly }: IQuickActionProps) => (
    <output data-quick-actions={issue.id}>{readOnly ? "read-only" : "writable"}</output>
  ),
}));

let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the cycle's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.CYCLE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  bound.drags.clear();
});
afterEach(async () => page.close());

const ended: Partial<Cycle> = { status: "completed" };
// nerve archives only ended cycles: a status other than completed shows that being archived alone makes it read-only
const archived: Partial<Cycle> = { status: "current", archived_at: "2026-12-01T09:00:00Z" };
const text = () => document.body.textContent ?? "";
const priorities = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((node) =>
    node.textContent?.startsWith("common.priority: ")
  );
/** What the shown layout offers: creation, inline edits of the row, dragging it, and its quick actions' mode. */
function offered() {
  const buttons = [...document.querySelectorAll<HTMLButtonElement>("button")];
  return {
    quickAdd: buttons.some((node) => ["issue.new", "issue.add.label"].includes(node.textContent?.trim() ?? "")),
    groupCreate: !!document.querySelector('button[aria-label^="create_work_item: "]'),
    inlineEdit: priorities().some((node) => !node.disabled),
    drag: [...bound.drags.values()].some((drag) => drag.canDrag?.({} as never) ?? true),
    quickActions: [...new Set([...document.querySelectorAll("[data-quick-actions]")].map((node) => node.textContent))],
  };
}
const none = { quickAdd: false, groupCreate: false, inlineEdit: false, drag: false, quickActions: ["read-only"] };
/** Each layout's writes the project page offers a member: its group "+", inline edits and drag. */
const writes: Record<Layout, { groupCreate: boolean; inlineEdit: boolean; drag: boolean }> = {
  list: { groupCreate: true, inlineEdit: true, drag: true },
  kanban: { groupCreate: true, inlineEdit: true, drag: true },
  spreadsheet: { groupCreate: false, inlineEdit: true, drag: false },
  calendar: { groupCreate: false, inlineEdit: false, drag: true },
};
const each = (Object.keys(layouts) as Layout[]).flatMap((layout) => [
  { layout, cycle: "ended", fields: ended },
  { layout, cycle: "archived", fields: archived },
]);

it.each(each)("the $layout of a cycle $cycle is read-only for a member", async ({ layout, fields }) => {
  await opened({ cycle: fields });
  await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
  expect(document.querySelectorAll('[id^="issue-first"]').length).toBeGreaterThan(0);
  expect(offered()).toEqual(none);
});

it.each(Object.keys(layouts) as Layout[])(
  "the %s of a cycle under way offers a member each write the project page does",
  async (layout) => {
    await opened();
    await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
    expect(offered()).toEqual({ quickAdd: true, ...writes[layout], quickActions: ["writable"] });
  }
);

it("shows a guest the cycle's work items without a write", async () => {
  await opened({ role: 5 });
  await page.show(cyclePreferencesOf(layouts.list.display), layouts.list.page());
  expect(document.querySelectorAll('[id^="issue-first"]').length).toBeGreaterThan(0);
  expect(offered()).toEqual(none);
});

it("offers a member of a cycle under way a new work item of the cycle when it has none", async () => {
  await opened();
  await page.show(cyclePreferencesOf(layouts.list.display), layouts.list.page(false));
  expect(text()).toContain("project_empty_state.cycle_work_items.title");
  const create = buttonOf("project_empty_state.cycle_work_items.cta_primary");
  expect(create?.disabled).toBe(false);
  await act(async () => create?.click());
  expect(page.app.commandPalette.isCreateIssueModalOpen).toBe(true);
  expect(page.app.commandPalette.createIssueStoreType).toBe(EIssuesStoreType.CYCLE);
});

it("offers a guest no new work item when the cycle has none", async () => {
  await opened({ role: 5 });
  await page.show(cyclePreferencesOf(layouts.list.display), layouts.list.page(false));
  expect(text()).toContain("project_empty_state.cycle_work_items.title");
  expect(buttonOf("project_empty_state.cycle_work_items.cta_primary")).toBeUndefined();
});

it.each([
  { cycle: "ended", fields: ended, filters: {} },
  { cycle: "ended", fields: ended, filters: { priority: ["high" as const] } },
  { cycle: "archived", fields: archived, filters: {} },
  { cycle: "archived", fields: archived, filters: { priority: ["high" as const] } },
])("only says why a cycle $cycle has no work items, with filters $filters", async ({ fields, filters }) => {
  await opened({ cycle: fields });
  await page.show(cyclePreferencesOf(layouts.list.display, filters), layouts.list.page(false));
  expect(text()).toContain("project_cycles.empty_state.completed_no_issues.title");
  expect(buttonOf("project_empty_state.cycle_work_items.cta_primary")).toBeUndefined();
  expect(buttonOf("project_issues.empty_state.issues_empty_filter.secondary_button.text")).toBeUndefined();
});
