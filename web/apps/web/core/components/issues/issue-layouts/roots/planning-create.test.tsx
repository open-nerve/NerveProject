// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A new work item's cycle and modules in its form, on a cycle's or a module's page (M6 P7a spec 2.6, R2; re-72 F2,
// F7). The form starts with the page's cycle or module while a creation sent now would join it, as the modal starts it
// before the form mounts; as it opens with its project it reads the project's two lists, its fields loading until both
// answer, the page's value held. What the form holds is what it sends, in one request: another cycle, No cycle, the
// page's module taken out or another added; a changed project clears both, and a field its project's switch hides
// sends nothing. Once nerve confirms a work item in the page's cycle or module the page reads its progress again, and
// not otherwise. An edit shows its work item's own modules, an archived one by its name and caption. The page's header,
// the app's modals, the form and its fields, the rows, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Issue, Project } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { CycleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header";
import { ModuleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { listIssue } from "@/store/issue/fake-issues";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  M1,
  choose,
  creates,
  cyclePreferencesOf,
  form,
  headerButton,
  layouts,
  made,
  modulePreferencesOf,
  planningPageTab,
  projectField,
  rowShown,
  send,
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
// jsdom has no viewport geometry, dialog transitions, editor or font loading; the form and its fields are real.
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
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
// The crumbs and the page's own quick actions have their own tests (page.test.tsx, actions-host.test.tsx).
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/cycles/quick-actions", () => ({ CycleQuickActions: () => null }));
vi.mock("@/components/modules/quick-actions", () => ({ ModuleQuickActions: () => null }));

const { CYCLE, MODULE } = EIssuesStoreType;
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const M2 = "019a0000-0000-7000-8000-0000000000d2";
const GONE = "019a0000-0000-7000-8000-0000000000d9";

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => page.close());

/** Each page: its route, its header, its preferences. */
const pages: Record<
  PlanningPageStoreType,
  { params: Record<string, string>; header: ReactNode; preferences: typeof cyclePreferencesOf }
> = {
  [CYCLE]: {
    params: { cycleId: C1 },
    header: <CycleIssuesHeader workspaceSlug="ws" projectId="A" cycleId={C1} />,
    preferences: cyclePreferencesOf,
  },
  [MODULE]: {
    params: { moduleId: M1 },
    header: <ModuleIssuesHeader workspaceSlug="ws" projectId="A" moduleId={M1} />,
    preferences: modulePreferencesOf,
  },
};
/** A tab at the page of type, as planningPageTab, as the fields say: a member's, of project A unless it says. */
async function opened(type: PlanningPageStoreType, fields: Parameters<typeof planningPageTab>[1] = {}) {
  bound.params = { workspaceSlug: "ws", projectId: "A", ...pages[type].params };
  page = await planningPageTab(type, { role: 15, ...fields });
  bound.app = page.app;
}
/** The page's header and the app's modals, before its work items, as the app puts them. */
const around = (type: PlanningPageStoreType) => (
  <>
    <div data-header>{pages[type].header}</div>
    <WorkItemLevelModals workItemIdentifier={undefined} />
  </>
);
/** Shows the page of type, its one work item (the page's, or row) listed, in the list. */
async function shown(type: PlanningPageStoreType, row?: Issue) {
  const list = row ? layouts.list.page(true, row) : page.listOf("list");
  await page.show(pages[type].preferences(layouts.list.display), list, around(type));
}
/** Opens the page header's new work item, whose form keeps the page's project. */
const fromHeader = () => act(async () => headerButton("issue.add.label")?.click());
/** Opens the app's own new work item, of no page, whose form lets the project be changed. */
const fromApp = () => act(async () => page.app.commandPalette.toggleCreateIssueModal(true));

/** The form's field of name: its dropdown's button, named by the field and its value. */
const fieldOf = (name: string) =>
  [...(form()?.querySelectorAll<HTMLButtonElement>("button") ?? [])].find((node) =>
    node.textContent?.startsWith(`${name}: `)
  );
const cycleField = () => fieldOf("common.cycle");
const moduleField = () => fieldOf("common.modules");
/** Whether the field shows loading. */
const loading = (field: HTMLButtonElement | undefined) => field?.querySelector('[role="status"]') != null;
/** Opens the field's list and picks its option of name, as a pointer does. */
async function pick(field: () => HTMLButtonElement | undefined, name: string) {
  await act(async () => {
    field()!.click();
    await vi.advanceTimersByTimeAsync(20);
  });
  const option = () =>
    [...document.querySelectorAll<HTMLElement>('[role="option"]')].find((node) => node.textContent === name);
  await until(() => option() !== undefined, `the option ${name}`);
  await act(async () => {
    option()!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

/** The reads of the project's list of a domain that is not archived (path), and of its archive. */
const listReads = (path: string, archived = false) =>
  page.nerve.to(path).filter((call) => call.method === "GET" && (call.query.archived === "true") === archived);
/**
 * The project's lists as nerve gives them: its cycles, the page's, Now (as cycle says, listed while it is the project's
 * and not archived), and Next; its modules, the page's, Core, and Edge; its archived modules, Gone.
 */
const listed = (project: Project, cycle: Partial<Cycle> = {}) => {
  const now = cycleOf(project, "Now", { id: C1, status: "current", ...cycle });
  const listedNow = now.project_id === project.id && now.archived_at === null ? [rowOf(now)] : [];
  return {
    cycles: [...listedNow, rowOf(cycleOf(project, "Next", { id: C2, status: "upcoming" }))],
    modules: [moduleRowOf(moduleOf(project, "Core", { id: M1 })), moduleRowOf(moduleOf(project, "Edge", { id: M2 }))],
    archivedModules: [
      moduleRowOf(moduleOf(project, "Gone", { id: GONE, status: "completed", archived_at: "2026-12-01T09:00:00Z" })),
    ],
  };
};
/** From now on nerve answers each read of project A's lists as listed, the page's cycle as cycle says. */
function answeringLists(cycle: Partial<Cycle> = {}) {
  const normal = page.nerve.respond;
  const lists = listed(page.project, cycle);
  const answers: [string, boolean, unknown[]][] = [
    [cycleRequests.list(page.project), false, lists.cycles],
    [moduleRequests.list(page.project), false, lists.modules],
    [moduleRequests.list(page.project), true, lists.archivedModules],
    [cycleRequests.list(page.project), true, []],
  ];
  page.nerve.respond = (call: Call) => {
    const answer = answers.find(
      ([path, archived]) => call.method === "GET" && call.path === path && (call.query.archived === "true") === archived
    );
    if (answer) call.answer(json(200, { data: answer[2] }));
    else normal?.(call);
  };
}
/** The reads of the progress of the page's cycle or module. */
const progressReads = (type: PlanningPageStoreType) =>
  page.nerve.to(type === CYCLE ? cycleRequests.progress({ id: C1 }) : moduleRequests.progress({ id: M1 }));
/** Waits a while for what would follow nerve's answer, then the page's progress reads. */
async function progressAfterwards(type: PlanningPageStoreType) {
  await act(async () => vi.advanceTimersByTimeAsync(100));
  return progressReads(type);
}

it("reads the project's two lists as the form opens; until both answer its fields show loading, holding the page's cycle", async () => {
  await opened(CYCLE);
  await shown(CYCLE);
  const cycles = () => listReads(cycleRequests.list(page.project));
  const modules = () => listReads(moduleRequests.list(page.project));
  expect([cycles(), modules()], "the lists, before the form opens").toEqual([[], []]);
  await fromHeader();
  await until(() => cycles().length === 1 && modules().length === 1, "the reads of the two lists");
  expect([loading(cycleField()), loading(moduleField())], "the fields, before the lists answer").toEqual([true, true]);
  expect(cycleField()?.textContent).toContain("Now");
  const lists = listed(page.project);
  await act(async () => {
    cycles()[0]!.answer(json(200, { data: lists.cycles }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect([loading(cycleField()), loading(moduleField())], "the fields, the modules still out").toEqual([true, true]);
  await act(async () => {
    modules()[0]!.answer(json(200, { data: lists.modules }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect([loading(cycleField()), loading(moduleField())], "the fields, once both answered").toEqual([false, false]);
  expect(cycleField()?.textContent).toBe("common.cycle: Now");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
  await send("#name", "Made", form);
  expect(creates()[0]!.body).toMatchObject({ name: "Made", cycle_id: C1 });
  expect(creates()[0]!.body).not.toHaveProperty("module_ids");
});

it("keeps the page's cycle in one POST; once nerve confirms the work item in it, the page reads its progress again", async () => {
  await opened(CYCLE);
  answeringLists();
  await shown(CYCLE);
  await fromHeader();
  await until(() => !loading(cycleField()), "the lists");
  await send("#name", "Made", form);
  expect(creates().map(({ body }) => body)).toEqual([expect.objectContaining({ name: "Made", cycle_id: C1 })]);
  await act(async () => creates()[0]!.answer(json(201, made("Made", { cycle_id: C1 }))));
  await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
  expect(await progressAfterwards(CYCLE)).toHaveLength(1);
  expect(creates()).toHaveLength(1);
});

it("sends another cycle picked in the form, as picked, in one POST; nerve's work item is not the page's, whose progress is not read", async () => {
  await opened(CYCLE);
  answeringLists();
  await shown(CYCLE);
  await fromHeader();
  await until(() => !loading(cycleField()), "the lists");
  await pick(cycleField, "Next");
  expect(cycleField()?.textContent).toBe("common.cycle: Next");
  await send("#name", "Elsewhere", form);
  expect(creates().map(({ body }) => body)).toEqual([expect.objectContaining({ name: "Elsewhere", cycle_id: C2 })]);
  await act(async () => creates()[0]!.answer(json(201, made("Elsewhere", { cycle_id: C2 }))));
  expect(toasts.map(({ type }) => type)).toEqual(["success"]);
  expect(await progressAfterwards(CYCLE)).toEqual([]);
  expect(document.querySelector('[id^="issue-made"]')).toBeNull();
  expect(creates()).toHaveLength(1);
});

it("sends no cycle_id once No cycle is picked in the form", async () => {
  await opened(CYCLE);
  answeringLists();
  await shown(CYCLE);
  await fromHeader();
  await until(() => !loading(cycleField()), "the lists");
  await pick(cycleField, "cycle.no_cycle");
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  await send("#name", "None", form);
  expect(creates()).toHaveLength(1);
  expect(creates()[0]!.body).toMatchObject({ name: "None" });
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
});

const ended: Partial<Cycle> = { status: "completed" };
// nerve archives only ended cycles: a status other than completed shows that being archived alone is enough
const archived: Partial<Cycle> = { status: "current", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { page: "of an ended cycle", fields: { cycle: ended } },
  { page: "of an archived cycle", fields: { cycle: archived } },
  { page: "whose cycle is held as another project's", fields: { cycle: { project_id: "B" } } },
])("starts the form with no cycle on the page $page, and sends none", async ({ fields }) => {
  await opened(CYCLE, fields);
  answeringLists(fields.cycle);
  await page.mount(around(CYCLE));
  await fromApp();
  await until(() => cycleField() !== undefined && !loading(cycleField()), "the form's cycle field");
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  await send("#name", "Plain", form);
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
});

it("shows no cycle field while the form's project has its cycles off, and sends none", async () => {
  await opened(CYCLE, { project: { cycle_view: false } });
  answeringLists();
  await page.mount(around(CYCLE));
  await fromApp();
  await until(() => moduleField() !== undefined && !loading(moduleField()), "the form's modules field");
  expect(cycleField()).toBeUndefined();
  await send("#name", "Plain", form);
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
});

it("sends no cycle once the form's project turns its cycles off, the field gone with the page's cycle in it", async () => {
  await opened(CYCLE);
  answeringLists();
  await shown(CYCLE);
  await fromApp();
  await until(() => !loading(cycleField()), "the lists");
  expect(cycleField()?.textContent).toBe("common.cycle: Now");
  await act(async () =>
    page.app.projectRoot.project.confirmProject(page.project, (held) => ({ ...held, cycle_view: false }))
  );
  expect(cycleField()).toBeUndefined();
  await send("#name", "Off", form);
  expect(creates()[0]!.body).not.toHaveProperty("cycle_id");
});

it("clears the form's cycle and modules as its project changes, and sends neither", async () => {
  await opened(CYCLE);
  const other = projectOf("B", page.workspace.id, { id: "B" });
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [page.project, other]);
  answeringLists();
  await shown(CYCLE);
  await fromApp();
  await until(() => !loading(cycleField()), "the lists");
  expect(cycleField()?.textContent).toBe("common.cycle: Now");
  await pick(projectField, "B");
  await until(() => fieldOf("common.project")?.textContent === "common.project: B", "the project B");
  expect(cycleField()?.textContent).toBe("common.cycle: cycle.label");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
  const elsewhere = () => page.nerve.to("/api/v0/projects/B/issues").filter((call) => call.method === "POST");
  await act(async () => {
    const input = form()!.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Moved");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => elsewhere().length === 1, "the creation in B");
  expect(elsewhere()[0]!.body).toMatchObject({ name: "Moved" });
  expect(elsewhere()[0]!.body).not.toHaveProperty("cycle_id");
  expect(elsewhere()[0]!.body).not.toHaveProperty("module_ids");
  expect(creates()).toEqual([]);
});

it("starts the form with the page's module; taken out of it, the one POST has no modules and the progress is not read", async () => {
  await opened(MODULE);
  answeringLists();
  await shown(MODULE);
  await fromHeader();
  await until(() => !loading(moduleField()), "the lists");
  expect(moduleField()?.textContent).toBe("common.modules: Core");
  await pick(moduleField, "Core");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
  await send("#name", "Loose", form);
  expect(creates()).toHaveLength(1);
  expect(creates()[0]!.body).toMatchObject({ name: "Loose" });
  expect(creates()[0]!.body).not.toHaveProperty("module_ids");
  await act(async () => creates()[0]!.answer(json(201, made("Loose", { state_id: "state" }))));
  expect(await progressAfterwards(MODULE)).toEqual([]);
});

it("sends the page's module and another added in the form, in one POST; nerve's work item is the page's, whose progress is read again", async () => {
  await opened(MODULE);
  answeringLists();
  await shown(MODULE);
  await fromHeader();
  await until(() => !loading(moduleField()), "the lists");
  await pick(moduleField, "Edge");
  expect(moduleField()?.textContent).toBe("common.modules: module.count: Core; Edge");
  await send("#name", "Both", form);
  expect(creates().map(({ body }) => body)).toEqual([expect.objectContaining({ name: "Both", module_ids: [M1, M2] })]);
  await act(async () => creates()[0]!.answer(json(201, made("Both", { state_id: "state", module_ids: [M1, M2] }))));
  await until(() => !!document.querySelector('[id^="issue-made"]'), "the new row");
  expect(await progressAfterwards(MODULE)).toHaveLength(1);
});

it("shows an edited work item's own modules, an archived one read for its name and captioned", async () => {
  await opened(MODULE);
  const row = listIssue("first", 1000, { module_ids: [M1, GONE], state_id: "state", priority: "high" });
  const preferences = modulePreferencesOf(layouts.list.display);
  answeringLists();
  // the row's own planning controls are off: only the dialog's field asks for the names
  await page.show(
    { ...preferences, display_properties: { ...preferences.display_properties, modules: false, cycle: false } },
    layouts.list.page(true, row),
    around(MODULE)
  );
  await until(rowShown, "the row");
  await choose("common.actions.edit");
  await until(
    () => moduleField()?.textContent === "common.modules: module.count: Core; Gone, common.archived",
    "the archived module's name"
  );
  expect(listReads(moduleRequests.list(page.project), true)).toHaveLength(1);
});
