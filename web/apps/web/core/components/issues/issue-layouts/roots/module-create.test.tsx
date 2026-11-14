// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A new work item of a module's page (P6b spec 2.5, 2.6, 2.7, R6). From the page's own entries (its header's new work
// item and its empty state's, whose form keeps the module's project, the quick add, a group's "+", whose form keeps it
// too) it is one request that joins the module; once nerve confirms it the page says so and lists it. The app's own
// new work item (the sidebar's, the command palette's, of no page) joins the module only while the page shows it taking
// work items; once the module is archived or the project's modules are off it creates as the project's page does. A
// refusal is said, never a success. The header, the app's modals, the form, the list, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { ModuleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import {
  M1,
  buttonOf,
  creates,
  form,
  globalCreate,
  headerButton,
  layouts,
  made,
  modulePreferencesOf,
  planningPageTab,
  projectField,
  quickAddForm,
  send,
  type Layout,
} from "./fake-planning-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
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
// jsdom has no viewport geometry, dialog transitions, editor or font loading; the form's own fields are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ...(await import("@/components/issues/issue-modal/fake-dialog")).modal,
}));
vi.mock(
  "@/components/editor/rich-text",
  async () => (await import("@/components/issues/issue-modal/fake-dialog")).editor
);
vi.mock("@/components/issues/issue-modal/components/default-properties", () => ({
  IssueDefaultProperties: () => null,
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
// P6a's crumbs and the module's quick actions have their own tests (page.test.tsx, actions-host.test.tsx).
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/modules/quick-actions", () => ({ ModuleQuickActions: () => null }));

let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the module's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.MODULE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  toasts.length = 0;
});
afterEach(async () => page.close());

/** The page's header and the app's modals, before its work items, as the app puts them. */
const around = (
  <>
    <div data-header>
      <ModuleIssuesHeader workspaceSlug="ws" projectId="A" moduleId={M1} />
    </div>
    <WorkItemLevelModals workItemIdentifier={undefined} />
  </>
);
/** Shows a member the page of the module as fields say, in the layout, its one work item grouped by state or none. */
async function shown(fields: Parameters<typeof planningPageTab>[1] = {}, layout: Layout = "list", rows = true) {
  await opened({ role: 15, ...fields });
  await page.show(modulePreferencesOf(layouts[layout].display), page.listOf(layout, rows), around);
}
/** Sends a new work item of name from the form that clicking opener opens, which keeps the module's project. */
async function fromLocked(opener: () => HTMLButtonElement | null | undefined, name: string) {
  await act(async () => opener()?.click());
  expect(projectField()?.disabled, "the form keeps the module's project").toBe(true);
  await send("#name", name, form);
}
const groupPlus = () => document.querySelector<HTMLButtonElement>('button[aria-label^="create_work_item: "]');
/** Each entry of the page, the layout it is in, whether the page lists a row, and how it sends a new work item. */
const entries: Record<string, { layout: Layout; rows: boolean; send: (name: string) => Promise<void> }> = {
  "the header's new work item": {
    layout: "list",
    rows: true,
    send: (name) => fromLocked(() => headerButton("issue.add.label"), name),
  },
  "the empty state's new work item": {
    layout: "list",
    rows: false,
    send: (name) => fromLocked(() => buttonOf("project_empty_state.module_work_items.cta_primary"), name),
  },
  "a list group's +": { layout: "list", rows: true, send: (name) => fromLocked(groupPlus, name) },
  "a kanban column's +": { layout: "kanban", rows: true, send: (name) => fromLocked(groupPlus, name) },
  "the quick add": {
    layout: "list",
    rows: true,
    send: async (name) => {
      await act(async () => buttonOf("issue.new")?.click());
      await send('input[name="name"]', name, quickAddForm);
    },
  },
};

it.each(Object.keys(entries))(
  "creates a work item of the module from %s in one request, says so and lists it",
  async (entry) => {
    await shown({}, entries[entry]!.layout, entries[entry]!.rows);
    await entries[entry]!.send("Made");
    expect(creates()[0]!.body).toMatchObject({ name: "Made", module_ids: [M1] });
    await act(async () => creates()[0]!.answer(json(201, made("Made", { state_id: "state", module_ids: [M1] }))));
    await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
    expect(toasts.map(({ type }) => type)).toEqual(["success"]);
    expect(creates()).toHaveLength(1);
  }
);

it.each([
  { entry: "the header's new work item", code: "planning.modules_disabled" },
  { entry: "the header's new work item", code: "planning.archived" },
  { entry: "the quick add", code: "planning.modules_disabled" },
  { entry: "the quick add", code: "planning.archived" },
])("says why nerve refused the work item of $entry ($code), never a success", async ({ entry, code }) => {
  await shown({}, entries[entry]!.layout);
  await entries[entry]!.send("Refused");
  expect(creates()[0]!.body).toMatchObject({ module_ids: [M1] });
  await act(async () => creates()[0]!.answer(problem(409, code)));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: `errors.${code.replace(".", "_")}` })]);
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
  expect(creates()).toHaveLength(1);
});

it("joins the module from the app's own new work item while the page shows it taking work items", async () => {
  await shown();
  await globalCreate("Global");
  expect(creates()[0]!.body).toEqual(expect.objectContaining({ name: "Global", module_ids: [M1] }));
  await act(async () => creates()[0]!.answer(json(201, made("Global", { state_id: "state", module_ids: [M1] }))));
  await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
});

// nerve archives only completed or cancelled modules: being archived alone is enough
const archived: Partial<Module> = { status: "completed", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { page: "of an archived module", fields: { module: archived } },
  { page: "while the project's modules are off", fields: { project: { module_view: false } } },
])("creates from the app's own new work item as the project's page does, on the page $page", async ({ fields }) => {
  await opened({ role: 15, ...fields });
  if (fields.project) await page.mount(around);
  else await page.show(modulePreferencesOf(layouts.list.display), page.listOf("list"), around);
  await globalCreate("Global");
  expect(creates()[0]!.body).toMatchObject({ name: "Global" });
  expect(creates()[0]!.body).not.toHaveProperty("module_ids");
  await act(async () => creates()[0]!.answer(json(201, made("Global"))));
  expect(toasts.map(({ type }) => type)).toEqual(["success"]);
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
});
