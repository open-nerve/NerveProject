// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A new work item of a cycle's page (P5b spec 2.5, 2.6, R4; re-66 M3, L7). From the page's own entries (its header's
// new work item, whose form keeps the cycle's project, the quick add, a group's "+", whose form keeps it too) it is
// one request that joins the cycle; once nerve confirms it the page says so and lists it. The app's own new work item
// (the sidebar's, the command palette's, of no page) joins the cycle only while the page shows it taking work items;
// once the cycle has ended, is archived or the project's cycles are off it creates as the project's page does. A
// refusal is said, never a success. The header, the app's modals, the form, the list, the stores and HTTP are real.
import { act, createElement, type ComponentProps, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { CycleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  buttonOf,
  creates,
  cycleField,
  form,
  globalCreate,
  headerButton,
  made,
  planningPageTab,
  projectField,
  cyclePreferencesOf,
  layouts,
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
// The form's default properties are out, but for the test that looks under the form's cycle field (P7a spec 2.6, R17):
// the real cycle and modules fields alone.
const properties = vi.hoisted(() => ({ planning: false }));
vi.mock("@/components/issues/issue-modal/components/default-properties", async () => {
  const { IssuePlanningFields } = await import("@/components/issues/issue-modal/components/planning-fields");
  type Props = Omit<ComponentProps<typeof IssuePlanningFields>, "loading"> & { planningLoading: boolean };
  return {
    IssueDefaultProperties: ({ planningLoading, ...props }: Props) =>
      properties.planning ? <IssuePlanningFields {...props} loading={planningLoading} /> : null,
  };
});
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
// P5a's crumbs and the cycle's quick actions have their own tests (page.test.tsx, actions-host.test.tsx).
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/cycles/quick-actions", () => ({ CycleQuickActions: () => null }));

let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the cycle's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.CYCLE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  properties.planning = false;
  toasts.length = 0;
});
afterEach(async () => page.close());

/** The page's header and the app's modals, before its work items, as the app puts them. */
const around = (
  <>
    <div data-header>
      <CycleIssuesHeader workspaceSlug="ws" projectId="A" cycleId={C1} />
    </div>
    <WorkItemLevelModals workItemIdentifier={undefined} />
  </>
);
/** Shows a member the page of the cycle as fields say, its one work item in the layout, grouped by state. */
async function shown(fields: Parameters<typeof planningPageTab>[1] = {}, layout: Layout = "list") {
  await opened({ role: 15, ...fields });
  await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page(), around);
}
/** Sends a new work item of name from the form a group's "+" opens, which keeps the cycle's project. */
async function fromGroup(name: string) {
  await act(async () => document.querySelector<HTMLButtonElement>('button[aria-label^="create_work_item: "]')?.click());
  expect(projectField()?.disabled, "the form keeps the cycle's project").toBe(true);
  await send("#name", name, form);
}
/** Each entry of the page, the layout it is in, and how it sends a new work item of name from the form it opens. */
const entries: Record<string, { layout: Layout; send: (name: string) => Promise<void> }> = {
  "the header's new work item": {
    layout: "list",
    send: async (name) => {
      await act(async () => headerButton("issue.add.label")?.click());
      expect(projectField()?.disabled, "the form keeps the cycle's project").toBe(true);
      await send("#name", name, form);
    },
  },
  "a list group's +": { layout: "list", send: fromGroup },
  "a kanban column's +": { layout: "kanban", send: fromGroup },
  "the quick add": {
    layout: "list",
    send: async (name) => {
      await act(async () => buttonOf("issue.new")?.click());
      await send('input[name="name"]', name, quickAddForm);
    },
  },
};

it.each(Object.keys(entries))(
  "creates a work item of the cycle from %s in one request, says so and lists it",
  async (entry) => {
    await shown({}, entries[entry]!.layout);
    await entries[entry]!.send("Made");
    expect(creates()[0]!.body).toMatchObject({ name: "Made", cycle_id: C1 });
    await act(async () => creates()[0]!.answer(json(201, made("Made", { cycle_id: C1 }))));
    await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
    expect(toasts.map(({ type }) => type)).toEqual(["success"]);
    expect(creates()).toHaveLength(1);
  }
);

it.each([
  { entry: "the header's new work item", code: "planning.cycles_disabled" },
  { entry: "the quick add", code: "planning.cycle_completed" },
  { entry: "the quick add", code: "planning.cycles_disabled" },
])("says why nerve refused the work item of $entry ($code), never a success", async ({ entry, code }) => {
  await shown({}, entries[entry]!.layout);
  await entries[entry]!.send("Refused");
  expect(creates()[0]!.body).toMatchObject({ cycle_id: C1 });
  await act(async () => creates()[0]!.answer(problem(409, code)));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: `errors.${code.replace(".", "_")}` })]);
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
  expect(creates()).toHaveLength(1);
});

it("shows under the cycle field of the header's new work item that nerve refused it as the cycle has ended (planning.cycle_completed), the field taking the focus, never a success", async () => {
  properties.planning = true;
  await shown();
  await entries["the header's new work item"]!.send("Refused");
  expect(creates()[0]!.body).toMatchObject({ cycle_id: C1 });
  await act(async () => creates()[0]!.answer(problem(409, "planning.cycle_completed")));
  const field = cycleField();
  expect(field?.getAttribute("aria-invalid")).toBe("true");
  expect(document.getElementById(field?.getAttribute("aria-describedby") ?? "")?.textContent).toBe(
    "errors.planning_cycle_completed"
  );
  expect(document.activeElement, "the focus").toBe(field);
  expect(toasts).toEqual([]);
  expect(form()).not.toBeNull();
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
  expect(creates()).toHaveLength(1);
});

it("joins the cycle from the app's own new work item while the page shows it taking work items", async () => {
  await shown();
  await globalCreate("Global");
  expect(creates()[0]!.body).toEqual(expect.objectContaining({ name: "Global", cycle_id: C1 }));
  await act(async () => creates()[0]!.answer(json(201, made("Global", { cycle_id: C1 }))));
  await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
});

const ended: Partial<Cycle> = { status: "completed" };
// nerve archives only ended cycles: a status other than completed shows that being archived alone is enough
const archived: Partial<Cycle> = { status: "current", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { page: "of an ended cycle", fields: { cycle: ended } },
  { page: "of an archived cycle", fields: { cycle: archived } },
  { page: "while the project's cycles are off", fields: { project: { cycle_view: false } } },
])("creates from the app's own new work item as the project's page does, on the page $page", async ({ fields }) => {
  await opened({ role: 15, ...fields });
  if (fields.project) await page.mount(around);
  else await page.show(cyclePreferencesOf(layouts.list.display), layouts.list.page(), around);
  await globalCreate("Global");
  expect(creates()[0]!.body).toMatchObject({ name: "Global" });
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
  await act(async () => creates()[0]!.answer(json(201, made("Global"))));
  expect(toasts.map(({ type }) => type)).toEqual(["success"]);
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
});
