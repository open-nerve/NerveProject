// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A work item's edit, with its cycle and modules in its form (M6 P7a spec 2.6, R3, R17). The form shows the work item's
// own cycle and modules, an ended cycle and an archived module too, never the page's. What the form changed goes: the
// cycle as one field, the modules as those the form took out and added since it opened, applied in the write's turn to
// the set the work item then holds, so that a module it joined elsewhere as the dialog read it stays, whatever its
// caller gives the dialog since; left as they were, neither goes. nerve's refusal of the cycle or of a module shows
// under its field, which takes the focus; one that names neither is said. The page, the row's dialog, the form and its
// fields, the stores and HTTP are real.
import { act, createElement, useState, type ReactNode } from "react";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, Project } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  M1,
  answeringPlanningLists,
  choose,
  cycleField,
  cyclePreferencesOf,
  form,
  layouts,
  moduleField,
  modulePreferencesOf,
  pick,
  planningPageTab,
  rowShown,
  type PlanningLists,
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

const { CYCLE, MODULE } = EIssuesStoreType;
const OLD = "019a0000-0000-7000-8000-0000000000c0";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const M2 = "019a0000-0000-7000-8000-0000000000d2";
const M3 = "019a0000-0000-7000-8000-0000000000d3";
const GONE = "019a0000-0000-7000-8000-0000000000d9";
const LATER = "2026-12-31T13:00:00Z";

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
});
afterEach(async () => page.close());

/** Each page: its route, its preferences. */
const pages: Record<PlanningPageStoreType, { params: Record<string, string>; preferences: typeof cyclePreferencesOf }> =
  {
    [CYCLE]: { params: { cycleId: C1 }, preferences: cyclePreferencesOf },
    [MODULE]: { params: { moduleId: M1 }, preferences: modulePreferencesOf },
  };
/**
 * The project's lists as nerve gives them: its cycles Now (the cycle page's), Next, and Old, which has ended; its
 * modules Core (the module page's), Edge and Side; its archived module Gone.
 */
const listed = (project: Project): PlanningLists => ({
  cycles: [
    rowOf(cycleOf(project, "Now", { id: C1, status: "current" })),
    rowOf(cycleOf(project, "Next", { id: C2, status: "upcoming" })),
    rowOf(cycleOf(project, "Old", { id: OLD, status: "completed" })),
  ],
  modules: [
    moduleRowOf(moduleOf(project, "Core", { id: M1 })),
    moduleRowOf(moduleOf(project, "Edge", { id: M2 })),
    moduleRowOf(moduleOf(project, "Side", { id: M3 })),
  ],
  archivedModules: [
    moduleRowOf(moduleOf(project, "Gone", { id: GONE, status: "completed", archived_at: "2026-12-01T09:00:00Z" })),
  ],
});
/** The page's work item "first" as nerve lists it, in the cycle and the modules fields say. */
const workItem = (fields: Partial<Issue>): Issue =>
  listIssue("first", 1000, { state_id: "state", priority: "high", ...fields });
/** The work item "first" as nerve reads it, with its description. */
const detailOf = (fields: Partial<Issue>): IssueDetail => ({ ...issueDetailOf("first"), ...workItem(fields) });

/**
 * Shows the page of type to a member, nerve listing its one work item "first" in the cycle and the modules fields say,
 * after what the test puts before the page; nerve answers each read of the project's lists.
 */
async function shown(type: PlanningPageStoreType, fields: Partial<Issue>, before?: ReactNode) {
  bound.params = { workspaceSlug: "ws", projectId: "A", ...pages[type].params };
  page = await planningPageTab(type, { role: 15 });
  bound.app = page.app;
  answeringPlanningLists(listed(page.project));
  await page.show(pages[type].preferences(layouts.list.display), layouts.list.page(true, workItem(fields)), before);
}
/** The reads of the work item itself: the dialog's, then the checks that follow a change of it. */
const reads = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "GET");
/**
 * From now on nerve answers the reads a dialog makes as the work item now is (fields: one it joined elsewhere, say): its
 * own, its resources', and the page's list's again once a change of it took the row off.
 */
function answeringReads(fields: Partial<Issue>) {
  const normal = page.nerve.respond;
  page.nerve.respond = (call: Call) => {
    if (call.method !== "GET") normal?.(call);
    else if (call.path === "/api/v0/issues/first") call.answer(json(200, detailOf(fields)));
    else if (/\/issues\/first\/(sub-issues|relations|links|reactions|activities|comments)$/.test(call.path))
      call.answer(json(200, { data: [], next_cursor: null }));
    else if (call.path === "/api/v0/projects/A/issues")
      call.answer(json(200, layouts.list.page(true, workItem(fields))));
    else normal?.(call);
  };
}
/** Waits for the open form to be ready to send: its work item read, its fields named. */
async function ready() {
  await until(() => form()?.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled === false, "the form");
  await act(async () => vi.advanceTimersByTimeAsync(100));
}
/** Opens the row's edit; nerve answers its reads of the work item as fields say it now is. */
async function edit(fields: Partial<Issue>) {
  await until(rowShown, "the row");
  answeringReads(fields);
  await choose("common.actions.edit");
  await ready();
}
const patches = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "PATCH");
/** Sends the open form; nerve holds the change it sends. */
async function sent() {
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await until(() => patches().length === 1, "the change");
  return patches()[0]!;
}
/** Types name as the work item's title in the open form. */
const titled = (name: string) =>
  act(async () => {
    const input = form()!.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, name);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
/** What a field's button says of nerve's refusal: whether it is refused, and the message that describes it. */
const refusedAs = (field: HTMLButtonElement | undefined) => ({
  invalid: field?.getAttribute("aria-invalid") ?? null,
  message: (field?.getAttribute("aria-describedby")?.split(/\s+/) ?? [])
    .map((id) => document.getElementById(id)?.textContent ?? "")
    .join(" "),
});
/** nerve's refusal of what it was sent, naming the field at path. */
const fieldRefused = (path: string) =>
  json(422, {
    status: 422,
    code: "validation_failed",
    title: "",
    errors: [{ field: path, code: "not_allowed", message: "" }],
  });

it.each<{ page: PlanningPageStoreType; of: string; held: string; fields: Partial<Issue>; shows: string[] }>([
  {
    page: CYCLE,
    of: "cycle",
    held: "in an ended cycle and an archived module",
    fields: { cycle_id: OLD, module_ids: [GONE] },
    shows: ["common.cycle: Old", "common.modules: Gone, common.archived"],
  },
  {
    page: CYCLE,
    of: "cycle",
    held: "in no cycle and no module",
    fields: { cycle_id: null, module_ids: [] },
    shows: ["common.cycle: cycle.label", "common.modules: module.no_module"],
  },
  {
    page: MODULE,
    of: "module",
    held: "in an ended cycle and an archived module",
    fields: { cycle_id: OLD, module_ids: [GONE] },
    shows: ["common.cycle: Old", "common.modules: Gone, common.archived"],
  },
])(
  "shows on the page of another $of the edited work item's own cycle and modules, $held, never the page's",
  async ({ page: type, fields, shows }) => {
    await shown(type, fields);
    await edit(fields);
    expect([cycleField()?.textContent, moduleField()?.textContent]).toEqual(shows);
  }
);

it("takes out of the work item the module taken out in the form, in its turn: one it joined elsewhere as the dialog read it stays", async () => {
  await shown(MODULE, { module_ids: [M1, GONE] });
  await edit({ module_ids: [M1, GONE, M2], updated_at: LATER });
  expect(moduleField()?.textContent, "the form, as it opened").toBe(
    "common.modules: module.count: Core; Gone, common.archived"
  );
  await pick(moduleField, "Core");
  expect(moduleField()?.textContent).toBe("common.modules: Gone, common.archived");
  const change = await sent();
  expect(change.body).toMatchObject({ module_ids: [GONE, M2] });
  expect(change.body).not.toHaveProperty("cycle_id");
  await act(async () => change.answer(json(200, detailOf({ module_ids: [GONE, M2], updated_at: LATER }))));
  expect(toasts.map(({ type }) => type)).toEqual(["success"]);
  expect(patches()).toHaveLength(1);
});

it("adds to the work item the module added in the form, in its turn, beside the one it joined elsewhere", async () => {
  await shown(MODULE, { module_ids: [M1, GONE] });
  await edit({ module_ids: [M1, GONE, M2], updated_at: LATER });
  await pick(moduleField, "Side");
  const change = await sent();
  expect(change.body).toMatchObject({ module_ids: [M1, GONE, M2, M3] });
  expect(change.body).not.toHaveProperty("cycle_id");
});

it.each([
  { picked: "Next", cycle: C2 },
  { picked: "cycle.no_cycle", cycle: null },
])(
  "changes the work item's cycle as one field ($picked), its modules left as they were not sent",
  async ({ picked, cycle }) => {
    await shown(MODULE, { cycle_id: C1, module_ids: [M1, GONE] });
    await edit({ cycle_id: C1, module_ids: [M1, GONE, M2], updated_at: LATER });
    await pick(cycleField, picked);
    const change = await sent();
    expect(change.body).toMatchObject({ cycle_id: cycle });
    expect(change.body).not.toHaveProperty("module_ids");
  }
);

it("sends what else the form changed, neither the cycle nor the modules it left as they were, though the work item joined a module elsewhere", async () => {
  await shown(MODULE, { cycle_id: C1, module_ids: [M1, GONE] });
  await edit({ cycle_id: C1, module_ids: [M1, GONE, M2], updated_at: LATER });
  await titled("Renamed");
  const change = await sent();
  expect(change.body).toMatchObject({ name: "Renamed" });
  expect(change.body).not.toHaveProperty("module_ids");
  expect(change.body).not.toHaveProperty("cycle_id");
});

/**
 * A caller that gives its edit's dialog the work item as the store holds it, each read changing what it gives; while a
 * check of it takes it off the store, as it last held it.
 */
const Following = observer(function Following() {
  const held = page.app.issue.issues.getIssueById("first");
  const [given, setGiven] = useState(held);
  if (held && held !== given) setGiven(held);
  return given ? (
    <CreateUpdateIssueModal isOpen data={given} storeType={EIssuesStoreType.PROJECT} onClose={() => {}} />
  ) : null;
});

it("works the modules' change out from the form as it opened, not from the work item its caller gives as a read changes it", async () => {
  await shown(MODULE, { module_ids: [M1, GONE] }, <Following />);
  await until(() => reads().length === 1, "the dialog's read of its work item");
  answeringReads({ module_ids: [M1, GONE, M2], updated_at: LATER });
  await act(async () => reads()[0]!.answer(json(200, detailOf({ module_ids: [M1, GONE, M2], updated_at: LATER }))));
  await ready();
  expect(page.app.issue.issues.getIssueById("first")?.module_ids, "what the caller gives").toEqual([M1, GONE, M2]);
  expect(moduleField()?.textContent, "the form, as it opened").toBe(
    "common.modules: module.count: Core; Gone, common.archived"
  );
  await pick(moduleField, "Core");
  const change = await sent();
  expect(change.body).toMatchObject({ module_ids: [GONE, M2] });
});

it("shows nerve's refusal of a module it was sent under the modules, which take the focus", async () => {
  await shown(MODULE, { module_ids: [M1, GONE] });
  await edit({ module_ids: [M1, GONE] });
  await pick(moduleField, "Side");
  const change = await sent();
  expect(change.body).toMatchObject({ module_ids: [M1, GONE, M3] });
  await act(async () => change.answer(fieldRefused("module_ids[1]")));
  expect(refusedAs(moduleField())).toEqual({ invalid: "true", message: "errors.field.not_allowed" });
  expect(document.activeElement, "the focus").toBe(moduleField());
  expect(refusedAs(cycleField()).invalid).toBeNull();
  expect(toasts).toEqual([]);
  expect(form()).not.toBeNull();
});

it.each([
  { refusal: "of its field", answer: () => fieldRefused("cycle_id"), message: "errors.field.not_allowed" },
  {
    refusal: "that it has ended",
    answer: () => problem(409, "planning.cycle_completed"),
    message: "errors.planning_cycle_completed",
  },
])(
  "shows nerve's refusal of the cycle it was sent, $refusal, under the cycle, which takes the focus",
  async ({ answer, message }) => {
    await shown(MODULE, { module_ids: [M1] });
    await edit({ module_ids: [M1] });
    await pick(cycleField, "Next");
    const change = await sent();
    await act(async () => change.answer(answer()));
    expect(refusedAs(cycleField())).toEqual({ invalid: "true", message });
    expect(document.activeElement, "the focus").toBe(cycleField());
    expect(refusedAs(moduleField()).invalid).toBeNull();
    expect(toasts).toEqual([]);
    expect(form()).not.toBeNull();
  }
);

it("says why nerve refused a module it was sent when nerve names no field (planning.archived), no field refused", async () => {
  await shown(MODULE, { module_ids: [M1] });
  await edit({ module_ids: [M1] });
  await pick(moduleField, "Side");
  const change = await sent();
  await act(async () => change.answer(problem(409, "planning.archived")));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.planning_archived" })]);
  expect([refusedAs(cycleField()).invalid, refusedAs(moduleField()).invalid]).toEqual([null, null]);
  expect(form()).not.toBeNull();
});
