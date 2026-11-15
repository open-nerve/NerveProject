// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A drag between the groups by cycle or module (P7b spec 2.7, 2.11; R3, R4) sends its request worked out in its turn:
// the cycle as one value (None's null), the modules as the set then held with the source left and the target joined, so
// an archived module stays and None leaves the source only; a board's cell sends both at once. A column nerve refuses
// sends nothing and says nerve's sentence; nerve's own refusal says its reason, the row staying where nerve has it; a
// cycle page reads its progress again. Hosts, cards, drag hook, stores and HTTP are real; the drag transport is not.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Issue, IssueListPage } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { TIssueGroupByOptions } from "@nerve/types";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { StoreContext } from "@/lib/store-context";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadArchivedCycles, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadArchivedModules, loadModules, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import { KanBan } from "./kanban/default";
import { KanBanSwimLanes } from "./kanban/swimlanes";
import { List } from "./list/default";

type Drop = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements>[0];
type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  params: {} as Record<string, string>,
  drops: new Map<Element, Drop>(),
  drags: new Map<Element, Drag>(),
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext(undefined),
  get rootStore() {
    return fixture.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { loginId: "drags" }, subscribe: () => () => {} },
  publicClient: {},
}));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "drags" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => fixture.params,
}));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  dropTargetForElements: (options: Drop) => {
    fixture.drops.set(options.element, options);
    return () => fixture.drops.delete(options.element);
  },
  draggable: (options: Drag) => {
    fixture.drags.set(options.element, options);
    return () => fixture.drags.delete(options.element);
  },
}));

class VisibleObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal("IntersectionObserver", VisibleObserver);
  vi.stubGlobal("ResizeObserver", VisibleObserver);
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.params = { workspaceSlug: "ws", projectId: "A" };
  fixture.drops.clear();
  fixture.drags.clear();
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/** nerve's moment of each answer to a write, each later than the one before. */
let answers = 0;
const later = () => `2026-10-29T00:00:0${++answers}Z`;
const archived_at = "2026-10-20T09:00:00Z";
/** The project's cycles and modules, an archived one of each in its archives: their ids by name. */
async function directory() {
  const { cycles, modules } = tab.app.planning;
  const cycle = (name: string, status: Cycle["status"]) => cycleOf(tab.project, name, { status });
  const mod = (name: string) => moduleOf(tab.project, name);
  const c = { sprint: cycle("Sprint", "current"), next: cycle("Next", "upcoming"), ended: cycle("Ended", "completed") };
  const m = { alpha: mod("Alpha"), beta: mod("Beta"), delta: mod("Delta") };
  const old = cycleOf(tab.project, "Old", { status: "completed", archived_at });
  const gamma = moduleOf(tab.project, "Gamma", { archived_at });
  const cycleRows = Object.values(c).map((each) => rowOf(each));
  const moduleRows = Object.values(m).map((each) => moduleRowOf(each));
  await loadCycles(tab.nerve, cycles, cycleRows, tab.project);
  await loadArchivedCycles(tab.nerve, cycles, [rowOf(old)], tab.project);
  await loadModules(tab.nerve, modules, moduleRows, tab.project);
  await loadArchivedModules(tab.nerve, modules, [moduleRowOf(gamma)], tab.project);
  return idsOf({ ...c, old, ...m, gamma });
}
const idsOf = <T extends Record<string, { id: string }>>(named: T) =>
  Object.fromEntries(Object.entries(named).map(([name, { id }]) => [name, id])) as Record<keyof T, string>;
type Ids = Awaited<ReturnType<typeof directory>>;
/** A group of nerve's answer: its key and its work items. */
const group = (key: string, data: Issue[]) => ({ key, total: data.length, data, next_cursor: null });
/** nerve's answer of these groups, its total the work items they hold. */
const answer = (...groups: ReturnType<typeof group>[]): IssueListPage => ({
  total: new Set(groups.flatMap((each) => each.data.map((issue) => issue.id))).size,
  groups,
});
/** nerve's answer grouped by cycle and by module: one cycle's group, its modules' work items its subgroups. */
function lanes(cycle: string, modules: Record<string, Issue[]>): IssueListPage {
  const subgroups = Object.entries(modules).map(([key, data]) => group(key, data));
  const { total } = answer(...subgroups);
  const subgroup_totals = subgroups.map((each) => ({ key: each.key, total: each.total }));
  return { total, groups: [{ key: cycle, total, subgroups }], subgroup_totals };
}
/** nerve's list of the project (or of the cycle page's cycle), grouped as query says, the page it answers. */
async function list(query: Record<string, string>, page: IssueListPage, cycle?: string) {
  const issues = tab.app.issue;
  const options = { limit: 10, query: { order_by: "sort_order" as const, ...query } };
  await answered(
    tab.nerve,
    () =>
      cycle
        ? issues.cycleIssues.fetchIssues("ws", { projectID: "A", entityID: cycle }, "init-loader", options)
        : issues.projectIssues.fetchIssues("ws", "A", "init-loader", options),
    ["GET", "/api/v0/projects/A/issues"],
    page,
    "the grouped list"
  );
  tab.nerve.calls.length = 0;
}

type Shown = { layout: "list" | "board" | "lanes"; grouping: TIssueGroupByOptions; lanes?: "module"; page?: boolean };
const { CYCLE, PROJECT } = EIssuesStoreType;
/** The page's host of its layout, its drops sent by the page's drag hook as the page's roots send them. */
const Host = observer(function Host({ layout, grouping, lanes: laneGrouping, page }: Shown) {
  const issues = page ? tab.app.issue.cycleIssues : tab.app.issue.projectIssues;
  const onDrop = useGroupIssuesDragNDrop(page ? CYCLE : PROJECT, "sort_order", grouping, laneGrouping);
  const common = {
    issuesMap: tab.app.issue.issues.issuesMap,
    groups: issues.groups!,
    orderBy: "sort_order" as const,
    updateIssue: undefined,
    quickActions: () => null,
    displayProperties: {},
    collapsedGroups: { group_by: [], sub_group_by: [] },
    canEditProperties: () => true,
    handleOnDrop: onDrop,
    showEmptyGroup: true,
    loadMoreIssues: () => {},
    handleCollapsedGroups: () => {},
    disableIssueCreation: true,
  };
  if (layout === "list") return <List {...common} enableIssueQuickAdd={false} group_by={grouping} />;
  const Board = layout === "lanes" ? KanBanSwimLanes : KanBan;
  const count = issues.getGroupIssueCount;
  const board = { enableQuickIssueCreate: false, getGroupIssueCount: count, sub_group_by: laneGrouping ?? null };
  return <Board {...common} {...board} group_by={grouping} />;
});
const show = (shown: Shown) =>
  act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>
          <MemoryRouter>
            <Host {...shown} />
          </MemoryRouter>
        </StoreContext.Provider>
      </SWRConfig>
    )
  );

const dataOf = (drop: Drop) =>
  drop.getData!({ element: drop.element, input: { clientX: 0, clientY: 0 } } as never) as Record<string, unknown>;
type At = [column: string, lane?: string];
/** The drop target of a group, or of a board's cell: its column's and its lane's. */
function target(...[column, lane]: At) {
  const found = [...fixture.drops.values()].filter((each) => {
    const data = dataOf(each);
    return data.type === "COLUMN" && data.groupId === column && data.subGroupId === lane;
  });
  expect(found, `the drop target of ${column} ${lane ?? ""}`).toHaveLength(1);
  return found[0]!;
}
/** Drags the card of work item id, as the user may, from its group (or cell) to the end of another one, and drops it. */
async function drag(id: string, from: At, to: At) {
  const source = target(...from);
  const card = [...fixture.drags.values()].find(
    (each) => source.element.contains(each.element) && each.getInitialData!({} as never).id === id
  );
  expect(card?.canDrag?.({} as never), `the card of ${id} can be dragged`).toBe(true);
  const destination = target(...to);
  await act(async () => {
    destination.onDrop!({
      source: { element: card!.element, data: card!.getInitialData!({} as never) },
      location: {
        initial: { dropTargets: [{ element: source.element, data: dataOf(source) }] },
        current: { dropTargets: [{ element: destination.element, data: dataOf(destination) }] },
      },
    } as never);
    await vi.advanceTimersByTimeAsync(1_000);
  });
}
/** The drag's request, the one PATCH nerve holds, of work item id. */
function sentDrag(id: string) {
  const patches = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(patches, "one PATCH").toHaveLength(1);
  expect(patches[0]!.path).toBe(`/api/v0/issues/${id}`);
  return patches[0]!;
}
/** nerve confirms the held request with the work item as its fields leave it. */
const confirm = (call: { answer: (response: Response) => void }, id: string, fields: Partial<Issue>) =>
  act(async () => {
    call.answer(json(200, { ...issueDetailOf(id), ...fields, updated_at: later() }));
    await vi.advanceTimersByTimeAsync(100);
  });
/** The rows of None and of a cycle's group as the page holds them, and their counts. */
const held = (cycle: string) => {
  const issues = tab.app.issue.projectIssues;
  const counts = ["none", cycle].map((key) => issues.getGroupIssueCount(key) ?? 0);
  return [issues.getIssueIds("none") ?? [], issues.getIssueIds(cycle) ?? [], ...counts];
};
const layouts = ["list", "board"] as const;

it.each(layouts)(
  "moves a work item from None into a cycle and back by drag in the %s, one request each; the groups follow nerve",
  async (layout) => {
    const ids = await directory();
    const s1 = listIssue("s1", 10, { cycle_id: ids.sprint });
    await list({ group_by: "cycle_id" }, answer(group(ids.sprint, [s1]), group("none", [listIssue("x", 1000)])));
    await show({ layout, grouping: "cycle" });
    await drag("x", ["none"], [ids.sprint]);
    const into = sentDrag("x");
    expect(into.body).toEqual({ cycle_id: ids.sprint, sort_order: 65545 });
    expect(held(ids.sprint)).toEqual([["x"], ["s1"], 1, 1]);
    await confirm(into, "x", { cycle_id: ids.sprint, sort_order: 65545 });
    expect(held(ids.sprint)).toEqual([[], ["s1", "x"], 0, 2]);
    tab.nerve.calls.length = 0;
    await drag("x", [ids.sprint], ["none"]);
    const back = sentDrag("x");
    expect(back.body).toEqual({ cycle_id: null, sort_order: 65535 });
    await confirm(back, "x", { cycle_id: null, sort_order: 65535 });
    expect(held(ids.sprint)).toEqual([["x"], ["s1"], 1, 1]);
    expect(toasts).toEqual([]);
  }
);

/** Each drag of x between modules, x in Alpha and the archived Gamma unless it comes from None: from, to, sent. */
const moduleDrags = ["to another, its archived one kept", "to None, out of the source only", "from None"] as const;
const moduleDrag = (ids: Ids): Record<(typeof moduleDrags)[number], [string, string, string[]]> => ({
  "to another, its archived one kept": [ids.alpha, ids.beta, [ids.gamma, ids.beta]],
  "to None, out of the source only": [ids.alpha, "none", [ids.gamma]],
  "from None": ["none", ids.beta, [ids.beta]],
});
it.each(layouts.flatMap((layout) => moduleDrags.map((name) => ({ layout, name }))))(
  "sends the modules x holds in its turn when dragged between modules $name in the $layout",
  async ({ layout, name }) => {
    const ids = await directory();
    const [from, to, sends] = moduleDrag(ids)[name];
    const x = listIssue("x", 1000, { module_ids: from === "none" ? [] : [ids.alpha, ids.gamma] });
    const groups = (from === "none" ? ["none"] : x.module_ids).map((key) => group(key, [x]));
    await list({ group_by: "module_id" }, answer(...groups));
    await show({ layout, grouping: "module" });
    await drag("x", [from], [to]);
    expect(sentDrag("x").body).toEqual({ module_ids: sends, sort_order: 65535 });
  }
);

it("works the modules out when the drag's turn comes, from the answer to the change before it", async () => {
  const ids = await directory();
  const x = listIssue("x", 1000, { module_ids: [ids.alpha, ids.gamma] });
  await list({ group_by: "module_id" }, answer(group(ids.alpha, [x]), group(ids.gamma, [x])));
  await show({ layout: "list", grouping: "module" });
  const scope = tab.app.issue.scope("ws", "A");
  const before = track(tab.app.issue.writes.changeIssue("x", { modules: { add: [ids.delta] } }, scope));
  await act(async () => until(() => tab.nerve.calls.length === 1, "the change before"));
  const first = tab.nerve.calls[0]!;
  expect(first.body).toEqual({ module_ids: [ids.alpha, ids.gamma, ids.delta] });
  await drag("x", [ids.alpha], [ids.beta]);
  expect(tab.nerve.calls, "the drag waits for its turn").toHaveLength(1);
  await confirm(first, "x", { module_ids: [ids.alpha, ids.gamma, ids.delta] });
  await act(async () => until(() => before.settled && tab.nerve.calls.length === 2, "the drag's turn"));
  expect(tab.nerve.calls[1]!.body).toEqual({ module_ids: [ids.gamma, ids.delta, ids.beta], sort_order: 65535 });
});

const byCycleAndModule = { group_by: "cycle_id", sub_group_by: "module_id" };
it("moves a board's cell's work item to another cycle and module in one request", async () => {
  const ids = await directory();
  const x = listIssue("x", 1000, { cycle_id: ids.sprint, module_ids: [ids.alpha, ids.gamma] });
  await list(byCycleAndModule, lanes(ids.sprint, { [ids.alpha]: [x], [ids.gamma]: [x] }));
  await show({ layout: "lanes", grouping: "cycle", lanes: "module" });
  await drag("x", [ids.sprint, ids.alpha], [ids.next, ids.beta]);
  expect(sentDrag("x").body).toEqual({ cycle_id: ids.next, module_ids: [ids.gamma, ids.beta], sort_order: 65535 });
});

/** Each column nerve refuses a work item's joining: its grouping, its key, nerve's sentence, whether the domains are off. */
const refusals = ["an ended cycle", "an archived cycle", "an archived module", "cycles off", "modules off"] as const;
type Refusal = [grouping: "cycle" | "module", key: string, says: string, off?: true];
const refusal = (ids: Ids): Record<(typeof refusals)[number], Refusal> => ({
  "an ended cycle": ["cycle", ids.ended, "errors.planning_cycle_completed"],
  "an archived cycle": ["cycle", ids.old, "errors.planning_archived"],
  "an archived module": ["module", ids.gamma, "errors.planning_archived"],
  "cycles off": ["cycle", ids.sprint, "errors.planning_cycles_disabled", true],
  "modules off": ["module", ids.alpha, "errors.planning_modules_disabled", true],
});
const warning = (message: string) => [{ type: "warning", title: "common.warning", message }];
it.each(layouts.flatMap((layout) => refusals.map((name) => ({ layout, name }))))(
  "sends no drop on a column nerve refuses ($name) in the $layout and says nerve's sentence",
  async ({ layout, name }) => {
    const ids = await directory();
    const [grouping, key, says, off] = refusal(ids)[name];
    if (off)
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
        projectOf("A", tab.workspace.id, { id: "A", cycle_view: false, module_view: false }),
      ]);
    const o = listIssue("o", 10, grouping === "cycle" ? { cycle_id: key } : { module_ids: [key] });
    await list({ group_by: `${grouping}_id` }, answer(group(key, [o]), group("none", [listIssue("x", 1000)])));
    await show({ layout, grouping });
    await drag("x", ["none"], [key]);
    expect(tab.nerve.calls).toEqual([]);
    expect(toasts).toEqual(warning(says));
    expect(tab.app.issue.projectIssues.getIssueIds("none")).toEqual(["x"]);
  }
);

it("sends no drop into a board's cell whose lane nerve refuses, and says so", async () => {
  const ids = await directory();
  const x = listIssue("x", 1000, { cycle_id: ids.sprint, module_ids: [ids.alpha] });
  const y = listIssue("y", 2000, { cycle_id: ids.sprint, module_ids: [ids.gamma] });
  await list(byCycleAndModule, lanes(ids.sprint, { [ids.alpha]: [x], [ids.gamma]: [y] }));
  await show({ layout: "lanes", grouping: "cycle", lanes: "module" });
  await drag("x", [ids.sprint, ids.alpha], [ids.sprint, ids.gamma]);
  expect(tab.nerve.calls).toEqual([]);
  expect(toasts).toEqual(warning("errors.planning_archived"));
});

it("reorders a work item within a column nerve refuses joining: only its order changes, which nerve takes", async () => {
  const ids = await directory();
  const ended = (id: string, key: number) => listIssue(id, key, { cycle_id: ids.ended });
  await list({ group_by: "cycle_id" }, answer(group(ids.ended, [ended("e1", 10), ended("e2", 20)])));
  await show({ layout: "board", grouping: "cycle" });
  await drag("e1", [ids.ended], [ids.ended]);
  expect(sentDrag("e1").body).toEqual({ sort_order: 65555 });
  expect(toasts).toEqual([]);
});

it("says nerve's reason when it refuses a drop the page let through; the row stays in nerve's group", async () => {
  const ids = await directory();
  await list({ group_by: "cycle_id" }, answer(group("none", [listIssue("x", 1000)])));
  await show({ layout: "list", grouping: "cycle" });
  await drag("x", ["none"], [ids.sprint]);
  await act(async () => {
    sentDrag("x").answer(problem(409, "planning.cycle_completed"));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycle_completed" }]);
  expect(held(ids.sprint)).toEqual([["x"], [], 1, 0]);
});

it("reads a cycle page's progress again, once, when nerve confirms a drag between its modules", async () => {
  const ids = await directory();
  fixture.params = { workspaceSlug: "ws", projectId: "A", cycleId: ids.sprint };
  const x = listIssue("x", 1000, { cycle_id: ids.sprint, module_ids: [ids.alpha, ids.gamma] });
  await list({ group_by: "module_id" }, answer(group(ids.alpha, [x]), group(ids.gamma, [x])), ids.sprint);
  await show({ layout: "list", grouping: "module", page: true });
  await drag("x", [ids.alpha], [ids.beta]);
  const call = sentDrag("x");
  expect(call.body).toEqual({ module_ids: [ids.gamma, ids.beta], sort_order: 65535 });
  const progress = () => tab.nerve.to(cycleRequests.progress({ id: ids.sprint }));
  expect(progress()).toEqual([]);
  await confirm(call, "x", { cycle_id: ids.sprint, module_ids: [ids.gamma, ids.beta] });
  expect(progress()).toHaveLength(1);
});
