// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The hosts of a grouping by cycle or module (P7b spec 2.4, 2.12): the list, the board and the board's lanes name the
// answer's groups through usePlanningNames, an archived one by its name and caption once its project's archive is
// read, one read a project whoever asks, none while the domain is off; a lane of an archived module is not lost, and a
// work item in two modules shows once in each.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueListPage } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadModules, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
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
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "groups" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "groups" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@/hooks/use-issue-layout-store", () => ({
  useIssueStoreType: () => EIssuesStoreType.PROJECT,
  useIssuesStore: () => ({ issues: tab.app.issue.projectIssues }),
}));
// Viewport geometry and the drag transport are the browser's; the rows show only their ids, by group.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("./list/blocks-list", () => ({
  IssueBlocksList: ({ issueIds, groupId }: { issueIds?: string[]; groupId: string }) => (
    <ul data-rows={groupId}>
      {issueIds?.map((id) => (
        <li key={id}>{id}</li>
      ))}
    </ul>
  ),
}));
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
  disableIssueCreation: true,
};
const listProps = { ...common, enableIssueQuickAdd: false, loadMoreIssues: () => {}, handleCollapsedGroups: () => {} };
const boardProps = {
  ...common,
  getGroupIssueCount: (group?: string, subgroup?: string, cumulative?: boolean) =>
    tab.app.issue.projectIssues.getGroupIssueCount(group, subgroup, cumulative),
  loadMoreIssues: () => {},
  handleCollapsedGroups: () => {},
};
const show = (node: ReactNode) =>
  act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>{node}</StoreContext.Provider>
      </SWRConfig>
    )
  );
async function list(query: { group_by: "cycle_id" | "module_id"; sub_group_by?: "module_id" }, page: IssueListPage) {
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
/** The project's reads of a domain's lists so far: {} for its list, archived=true for its archive. */
const reads = (path: string) =>
  tab.nerve.calls.filter((call) => call.method === "GET" && call.path === path).map((call) => call.query);
const labelled = (label: string) => host.querySelectorAll(`[aria-label="${label}"]`).length;

// Each host reads the names itself: one alone, then both, which ask for one read between them.
it.each<[string, ("list" | "board")[]]>([
  ["the list", ["list"]],
  ["the board", ["board"]],
  ["the list and the board", ["list", "board"]],
])("names an archived cycle's group in %s by one read of its project's archive", async (_, hosts) => {
  const sprint = cycleOf(tab.project, "Sprint", { status: "current" });
  const old = cycleOf(tab.project, "Old", { status: "completed", archived_at: archivedAt });
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(sprint)], tab.project);
  await list(
    { group_by: "cycle_id" },
    {
      total: 2,
      groups: [
        { key: old.id, total: 1, data: [listIssue("x", 10, { cycle_id: old.id })], next_cursor: null },
        { key: sprint.id, total: 1, data: [listIssue("y", 20, { cycle_id: sprint.id })], next_cursor: null },
        { key: "none", total: 0, data: [], next_cursor: null },
      ],
    }
  );
  const groups = tab.app.issue.projectIssues.groups!;
  await show(
    <>
      {hosts.includes("list") && <List {...listProps} groups={groups} group_by="cycle" />}
      {hosts.includes("board") && <KanBan {...boardProps} groups={groups} group_by="cycle" sub_group_by={null} />}
    </>
  );
  await act(async () => until(() => reads(cycleRequests.list(tab.project)).length > 0, "the archive's read"));
  // Before the archive answers, the group shows no name.
  expect(labelled("—")).toBe(hosts.length);
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(reads(cycleRequests.list(tab.project))).toEqual([{ archived: "true" }]);
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, { data: [rowOf(old)] }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(labelled("Old, common.archived")).toBe(hosts.length);
  expect(labelled("Sprint")).toBe(hosts.length);
  expect(reads(cycleRequests.list(tab.project))).toHaveLength(1);
});

it("reads no name of a group while the project has its cycles turned off: the group shows none", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    projectOf("A", tab.workspace.id, { id: "A", cycle_view: false }),
  ]);
  tab.nerve.calls.length = 0;
  const old = cycleOf(tab.project, "Old", { archived_at: archivedAt });
  await list(
    { group_by: "cycle_id" },
    { total: 1, groups: [{ key: old.id, total: 1, data: [listIssue("x", 10)], next_cursor: null }] }
  );
  const groups = tab.app.issue.projectIssues.groups!;
  await show(
    <>
      <List {...listProps} groups={groups} group_by="cycle" />
      <KanBan {...boardProps} groups={groups} group_by="cycle" sub_group_by={null} />
    </>
  );
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.nerve.calls).toEqual([]);
  expect(labelled("—")).toBe(2);
});

it("keeps an archived cycle's group and an archived module's lane, by name and caption, on a board's lanes", async () => {
  const sprint = cycleOf(tab.project, "Sprint", { status: "current" });
  const old = cycleOf(tab.project, "Old", { status: "completed", archived_at: archivedAt });
  const alpha = moduleOf(tab.project, "Alpha");
  const gamma = moduleOf(tab.project, "Gamma", { archived_at: archivedAt });
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(sprint)], tab.project);
  await loadModules(tab.nerve, tab.app.planning.modules, [moduleRowOf(alpha)], tab.project);
  const row = listIssue("x", 10, { cycle_id: old.id, module_ids: [gamma.id] });
  await list(
    { group_by: "cycle_id", sub_group_by: "module_id" },
    {
      total: 1,
      groups: [{ key: old.id, total: 1, subgroups: [{ key: gamma.id, total: 1, data: [row], next_cursor: null }] }],
      subgroup_totals: [{ key: gamma.id, total: 1 }],
    }
  );
  // The lanes collapsed show no board of their own: the lanes' host names both levels itself.
  await show(
    <KanBanSwimLanes
      {...boardProps}
      collapsedGroups={{ group_by: [], sub_group_by: [alpha.id, gamma.id, "none"] }}
      groups={tab.app.issue.projectIssues.groups!}
      group_by="cycle"
      sub_group_by="module"
      enableQuickIssueCreate={false}
    />
  );
  const archives = [cycleRequests.list(tab.project), moduleRequests.list(tab.project)];
  await act(async () => until(() => archives.every((path) => reads(path).length > 0), "the archives' reads"));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(archives.map(reads)).toEqual([[{ archived: "true" }], [{ archived: "true" }]]);
  await act(async () => {
    const answer = (path: string, rows: unknown[]) =>
      tab.nerve.calls.find((call) => call.path === path)!.answer(json(200, { data: rows }));
    answer(archives[0]!, [rowOf(old)]);
    answer(archives[1]!, [moduleRowOf(gamma)]);
    await vi.advanceTimersByTimeAsync(20);
  });
  const lanes = [...host.querySelectorAll("button[aria-expanded]")].map((button) => button.textContent);
  expect(lanes).toEqual(["Alpha0", "Gamma, common.archived1", "None0"]);
  expect(host.textContent).toContain("Sprint0");
  expect(host.textContent).toContain("Old, common.archived1");
});

it("shows a work item in two modules once in each module's group", async () => {
  const alpha = moduleOf(tab.project, "Alpha");
  const beta = moduleOf(tab.project, "Beta");
  await loadModules(
    tab.nerve,
    tab.app.planning.modules,
    [alpha, beta].map((item) => moduleRowOf(item)),
    tab.project
  );
  const row = listIssue("x", 10, { module_ids: [alpha.id, beta.id] });
  await list(
    { group_by: "module_id" },
    {
      total: 1,
      groups: [
        { key: alpha.id, total: 1, data: [row], next_cursor: null },
        { key: beta.id, total: 1, data: [row], next_cursor: null },
      ],
    }
  );
  await show(<List {...listProps} groups={tab.app.issue.projectIssues.groups!} group_by="module" />);
  const rows = (group: string) =>
    [...host.querySelectorAll(`[data-rows="${group}"] li`)].map((item) => item.textContent);
  expect(rows(alpha.id)).toEqual(["x"]);
  expect(rows(beta.id)).toEqual(["x"]);
  expect(tab.nerve.calls).toEqual([]);
});
