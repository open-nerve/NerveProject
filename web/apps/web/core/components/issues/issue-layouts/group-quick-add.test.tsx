// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A group's new work items by cycle or module (P7b spec 2.4, 2.5; R4, R12): a quick add in a column of a grouping by
// cycle or module sends the column's cycle or module beside the project's default state, None's nothing, a board's
// cell its column's and its lane's; a grouping by state gives its state as before. A column nerve refuses a work item's
// joining (an ended cycle, an archived cycle or module, any while the project has the domain turned off) offers
// neither its header's "+" nor a quick add, nor does a cell of a lane it refuses. The hosts, headers, quick add, stores
// and HTTP are real.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueCreate, IssueListPage } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { loadArchivedCycles, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadArchivedModules, loadModules, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf, stateOf } from "@/store/project/fake-projects";
import { KanBan } from "./kanban/default";
import { KanBanSwimLanes } from "./kanban/swimlanes";
import { List } from "./list/default";

const bound = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext(undefined),
  get rootStore() {
    return bound.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { loginId: "groups" }, subscribe: () => () => {} },
  publicClient: {},
}));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "groups" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@/hooks/use-issue-layout-store", () => ({
  useIssueStoreType: () => EIssuesStoreType.PROJECT,
  useIssuesStore: () => ({ issues: tab.app.issue.projectIssues }),
}));
vi.mock("../create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
// Viewport geometry and the drag transport are the browser's; the rows are not what is under test.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("./list/blocks-list", () => ({ IssueBlocksList: () => null }));
vi.mock("./kanban/blocks-list", () => ({ KanbanIssueBlocksList: () => null }));

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
  tab = await issueStoreTab();
  bound.app = tab.app;
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const archivedAt = "2026-10-20T09:00:00Z";
/** The project's directory: its default state Todo and Doing, its cycles and modules, archived ones in its archives. */
async function directory() {
  const states = [
    stateOf(tab.project, "Todo", "unstarted", 1, { default: true }),
    stateOf(tab.project, "Doing", "started", 2),
  ];
  await answered(
    tab.nerve,
    () => tab.app.state.fetchProjectStates("A"),
    ["GET", "/api/v0/projects/A/states"],
    { data: states },
    "the states"
  );
  const sprint = cycleOf(tab.project, "Sprint", { status: "current" });
  const ended = cycleOf(tab.project, "Ended", { status: "completed" });
  const old = cycleOf(tab.project, "Old", { status: "completed", archived_at: archivedAt });
  const alpha = moduleOf(tab.project, "Alpha");
  const gamma = moduleOf(tab.project, "Gamma", { archived_at: archivedAt });
  const { cycles, modules } = tab.app.planning;
  await loadCycles(tab.nerve, cycles, [rowOf(sprint), rowOf(ended)], tab.project);
  await loadArchivedCycles(tab.nerve, cycles, [rowOf(old)], tab.project);
  await loadModules(tab.nerve, modules, [moduleRowOf(alpha)], tab.project);
  await loadArchivedModules(tab.nerve, modules, [moduleRowOf(gamma)], tab.project);
  return { todo: states[0]!.id, doing: states[1]!.id, sprint: sprint.id, ended: ended.id, old: old.id, alpha, gamma };
}
/** nerve's list of the project, grouped as query says, the page it answers. */
async function list(query: Record<string, string>, page: IssueListPage) {
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { order_by: "sort_order", ...query },
      }),
    ["GET", "/api/v0/projects/A/issues"],
    page,
    "the grouped list"
  );
  tab.nerve.calls.length = 0;
}
/** A group of nerve's answer, of one work item. */
const group = (key: string, id = `in-${key}`) => ({ key, total: 1, data: [listIssue(id, 10)], next_cursor: null });

const create = (project: string | null | undefined, body: IssueCreate) =>
  project ? tab.app.issue.projectIssues.createIssue("ws", project, body) : Promise.resolve(undefined);
const common = {
  issuesMap: {},
  orderBy: "sort_order" as const,
  updateIssue: undefined,
  quickActions: () => null,
  displayProperties: undefined,
  collapsedGroups: { group_by: [], sub_group_by: [] },
  canEditProperties: () => true,
  handleOnDrop: async () => {},
  showEmptyGroup: true,
  quickAddCallback: create,
  loadMoreIssues: () => {},
  handleCollapsedGroups: () => {},
};
const boardProps = {
  ...common,
  enableQuickIssueCreate: true,
  getGroupIssueCount: (column?: string, lane?: string, cumulative?: boolean) =>
    tab.app.issue.projectIssues.getGroupIssueCount(column, lane, cumulative),
};
type Grouping = "state" | "cycle" | "module";
/** Shows the project's list grouped by grouping in the host, its groups as nerve answered them. */
const show = (layout: "list" | "board", grouping: Grouping) => {
  const groups = tab.app.issue.projectIssues.groups!;
  return act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>
          {layout === "list" ? (
            <List {...common} enableIssueQuickAdd groups={groups} group_by={grouping} />
          ) : (
            <KanBan {...boardProps} groups={groups} group_by={grouping} sub_group_by={null} />
          )}
        </StoreContext.Provider>
      </SWRConfig>
    )
  );
};
/** Sends a new work item named New from the quick add in at (the board's cell, else the page) and gives nerve's request. */
async function quickAdd(at: ParentNode = host) {
  const open = [...at.querySelectorAll("button")].filter((button) => button.textContent === "issue.new");
  expect(open, "one quick add").toHaveLength(1);
  await act(async () => open[0]!.click());
  const input = host.querySelector<HTMLInputElement>('input[name="name"]')!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => input.form!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await act(async () => until(() => tab.nerve.calls.length === 1, "the creation"));
  const call = tab.nerve.calls[0]!;
  expect([call.method, call.path]).toEqual(["POST", "/api/v0/projects/A/issues"]);
  await act(async () => {
    call.answer(json(201, issueDetailOf("new")));
    await vi.advanceTimersByTimeAsync(20);
  });
  return call.body;
}
/** A board's cell: its column's and its lane's ids. */
const cell = (column: string, lane?: string) => host.querySelector(`[id="${column}__${lane}"]`)!;
/** What the groups offer, in the page's order: each group's name, its header's "+" and a quick add. */
const offered = () =>
  [...host.querySelectorAll("button")].flatMap((button) => {
    const label = button.getAttribute("aria-label") ?? "";
    if (label.startsWith("create_work_item: ")) return ["+"];
    if (button.hasAttribute("aria-expanded")) return [label];
    return button.textContent === "issue.new" ? ["quick add"] : [];
  });

type Ids = Awaited<ReturnType<typeof directory>>;
/** Each group a quick add is in: its grouping, its key in nerve's answer and what the new work item is sent with. */
const cases: Record<string, { grouping: Grouping; key: (ids: Ids) => string; sends: (ids: Ids) => object }> = {
  "a cycle's": { grouping: "cycle", key: (ids) => ids.sprint, sends: (ids) => ({ cycle_id: ids.sprint }) },
  "None's of a grouping by cycle": { grouping: "cycle", key: () => "none", sends: () => ({}) },
  "a module's": { grouping: "module", key: (ids) => ids.alpha.id, sends: (ids) => ({ module_ids: [ids.alpha.id] }) },
  "None's of a grouping by module": { grouping: "module", key: () => "none", sends: () => ({}) },
  "a state's": { grouping: "state", key: (ids) => ids.doing, sends: (ids) => ({ state_id: ids.doing }) },
};
it.each((["list", "board"] as const).flatMap((layout) => Object.keys(cases).map((name) => ({ layout, name }))))(
  "sends the group's value from the quick add of $name group in the $layout, beside the default state",
  async ({ layout, name }) => {
    const ids = await directory();
    const { grouping, key, sends } = cases[name]!;
    await list({ group_by: `${grouping}_id` }, { total: 1, groups: [group(key(ids))] });
    await show(layout, grouping);
    const sent = await quickAdd(layout === "board" ? cell(key(ids)) : host);
    expect(sent).toEqual({ name: "New", state_id: ids.todo, ...sends(ids) });
  }
);

it("sends a board cell's cycle and module from its quick add, its column's and its lane's", async () => {
  const ids = await directory();
  await list(
    { group_by: "cycle_id", sub_group_by: "module_id" },
    {
      total: 1,
      groups: [{ key: ids.sprint, total: 1, subgroups: [group(ids.alpha.id)] }],
      subgroup_totals: [{ key: ids.alpha.id, total: 1 }],
    }
  );
  await act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>
          <KanBanSwimLanes
            {...boardProps}
            groups={tab.app.issue.projectIssues.groups!}
            group_by="cycle"
            sub_group_by="module"
          />
        </StoreContext.Provider>
      </SWRConfig>
    )
  );
  expect(await quickAdd(cell(ids.sprint, ids.alpha.id))).toEqual({
    name: "New",
    state_id: ids.todo,
    cycle_id: ids.sprint,
    module_ids: [ids.alpha.id],
  });
});

it.each((["list", "board"] as const).flatMap((layout) => [true, false].map((on) => ({ layout, on }))))(
  "offers no + and no quick add on a column nerve refuses, by cycle and by module, in the $layout, the domains on: $on",
  async ({ layout, on }) => {
    const ids = await directory();
    if (!on)
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
        projectOf("A", tab.workspace.id, { id: "A", cycle_view: false, module_view: false }),
      ]);
    const keys = [ids.sprint, ids.ended, ids.old, "none"];
    await list({ group_by: "cycle_id" }, { total: 4, groups: keys.map((key) => group(key)) });
    await show(layout, "cycle");
    const joins = on ? ["+", "quick add"] : [];
    expect(offered()).toEqual(["Sprint", ...joins, "Ended", "Old, common.archived", "None", "+", "quick add"]);
    await act(async () => view.render(null));
    await list(
      { group_by: "module_id" },
      { total: 3, groups: [ids.alpha.id, ids.gamma.id, "none"].map((key) => group(key)) }
    );
    await show(layout, "module");
    expect(offered()).toEqual(["Alpha", ...joins, "Gamma, common.archived", "None", "+", "quick add"]);
  }
);

it("offers a quick add in no cell of a board's lane or column nerve refuses", async () => {
  const ids = await directory();
  await list(
    { group_by: "cycle_id", sub_group_by: "module_id" },
    {
      total: 2,
      groups: [
        { key: ids.sprint, total: 1, subgroups: [group(ids.alpha.id)] },
        { key: ids.ended, total: 1, subgroups: [group(ids.gamma.id)] },
      ],
      subgroup_totals: [
        { key: ids.alpha.id, total: 1 },
        { key: ids.gamma.id, total: 1 },
      ],
    }
  );
  await act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>
          <KanBanSwimLanes
            {...boardProps}
            groups={tab.app.issue.projectIssues.groups!}
            group_by="cycle"
            sub_group_by="module"
          />
        </StoreContext.Provider>
      </SWRConfig>
    )
  );
  const pluses = [...host.querySelectorAll('button[aria-label^="create_work_item: "]')].map((button) =>
    button.getAttribute("aria-label")
  );
  expect(pluses).toEqual(["create_work_item: Sprint", "create_work_item: None"]);
  const cells = [...host.querySelectorAll('[id*="__"]')].filter((node) =>
    [...node.querySelectorAll("button")].some((button) => button.textContent === "issue.new")
  );
  expect(cells.map((node) => node.id)).toEqual([
    `${ids.sprint}__${ids.alpha.id}`,
    `none__${ids.alpha.id}`,
    `${ids.sprint}__none`,
    "none__none",
  ]);
});
