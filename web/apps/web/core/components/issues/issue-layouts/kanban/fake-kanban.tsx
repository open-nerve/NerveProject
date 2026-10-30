// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import type { ReactNode } from "react";
import type { TRenderQuickActions } from "../list/list-view-types";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { TIssueGroupByOptions } from "@nerve/types";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import type { Issue } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import { answered } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { stateOf } from "@/store/project/fake-projects";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { KanBan } from "./default";
import { KanBanSwimLanes } from "./swimlanes";

type Drop = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements>[0];
type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  drops: new Map<Element, Drop>(),
  drags: new Map<Element, Drag>(),
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext(undefined),
  get rootStore() {
    return fixture.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "A" } }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
// Only viewport geometry and the browser's drag transport are replaced; cards, payloads and writes are real.
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
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
export let tab: Awaited<ReturnType<typeof issueStoreTab>>;
export let container: HTMLDivElement;
export let view: ReturnType<typeof createRoot>;
export const dragSources = fixture.drags;
let quickActions: TRenderQuickActions;
let groupBy: TIssueGroupByOptions;
let subgroupBy: TIssueGroupByOptions | undefined;
export const first = listIssue("first", 1000, { state_id: "todo" });
export const a = listIssue("a", 2000, { state_id: "done" });
export const b = listIssue("b", 3000, { state_id: "done" });
export const c = listIssue("c", 4000, { state_id: "done" });
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  quickActions = () => null;
  groupBy = "state";
  subgroupBy = undefined;
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.drops.clear();
  fixture.drags.clear();
  await answered(
    tab.nerve,
    () => tab.app.state.fetchProjectStates("A"),
    ["GET", "/api/v0/projects/A/states"],
    {
      data: [
        stateOf(tab.project, "Todo", "unstarted", 1, { id: "todo" }),
        stateOf(tab.project, "Done", "completed", 2, { id: "done" }),
      ],
    },
    "states"
  );
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const Host = observer(function Host() {
  const issues = tab.app.issue.projectIssues;
  const onDrop = useGroupIssuesDragNDrop(EIssuesStoreType.PROJECT, "sort_order", groupBy, subgroupBy);
  const refresh = useIssueListRefresh(issues.currentList, []);
  const Component = subgroupBy ? KanBanSwimLanes : KanBan;
  return (
    <>
      <IssueListRetry {...refresh} />
      <Component
        issuesMap={tab.app.issue.issues.issuesMap}
        groups={issues.groups!}
        getGroupIssueCount={issues.getGroupIssueCount}
        displayProperties={{}}
        group_by={groupBy}
        sub_group_by={subgroupBy}
        orderBy="sort_order"
        updateIssue={undefined}
        quickActions={quickActions}
        collapsedGroups={{ group_by: [], sub_group_by: [] }}
        handleCollapsedGroups={() => {}}
        loadMoreIssues={(group, subgroup) => {
          void issues.fetchNextIssues("ws", "A", group, subgroup).catch(() => {});
        }}
        enableQuickIssueCreate={false}
        showEmptyGroup
        disableIssueCreation
        canEditProperties={() => true}
        handleOnDrop={onDrop}
      />
    </>
  );
});
export async function mount(rows: Issue[] = [a, b, c], actions?: TRenderQuickActions) {
  if (actions) quickActions = actions;
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        query: { order_by: "sort_order", group_by: "state_id" },
        limit: 3,
      }),
    ["GET", "/api/v0/projects/A/issues"],
    {
      total: rows.length + 1,
      groups: [
        { key: "todo", total: 1, data: [first], next_cursor: null },
        { key: "done", total: rows.length, data: rows, next_cursor: null },
      ],
    },
    "kanban"
  );
  await renderBoard();
  tab.nerve.calls.length = 0;
}
function column(group: string) {
  return [...fixture.drops.values()].find(
    (d) => d.getData?.({ element: d.element, input: { clientX: 0, clientY: 0 } } as never).groupId === group
  )!;
}
const columnTarget = (d: Drop) => ({ element: d.element, data: d.getData!({} as never) });
export function payload(
  sourceId: string,
  sourceGroup: string,
  targetId: string | undefined,
  targetGroup = "done",
  below = false
) {
  const drag = [...fixture.drags.values()].find((d) => d.getInitialData?.({} as never).id === sourceId)!;
  expect(drag, "the real card must register a drag source").toBeDefined();
  const sourceColumn = column(sourceGroup);
  const targetColumn = column(targetGroup);
  const card = [...fixture.drops.values()].find(
    (d) => d.getData?.({ input: { clientY: 0 }, element: d.element } as never).id === targetId && targetId
  )!;
  if (card) vi.spyOn(card.element, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 0, 100, 100));
  return {
    source: { data: drag.getInitialData!({} as never), element: drag.element },
    location: {
      initial: { dropTargets: [columnTarget(sourceColumn)] },
      current: {
        dropTargets: [
          ...(card
            ? [
                {
                  element: card.element,
                  data: card.getData!({
                    element: card.element,
                    input: { clientY: below ? 90 : 10, clientX: 50 },
                  } as never),
                },
              ]
            : []),
          columnTarget(targetColumn),
        ],
      },
    },
  };
}
export function visible(group: string) {
  return [...column(group).element.querySelectorAll('a[id^="issue_"]')].map((el) => el.id.split("_")[1]);
}
export async function send(sourceId: string, sourceGroup: string, targetId: string | undefined, below = false) {
  const data = payload(sourceId, sourceGroup, targetId, "done", below);
  await act(async () => {
    column("done").onDrop!(data as never);
    await until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "card PATCH");
  });
  return tab.nerve.calls.find((call) => call.method === "PATCH")!;
}

export async function renderBoard() {
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <MemoryRouter>
          <Host />
        </MemoryRouter>
      </StoreContext.Provider>
    )
  );
}
export function setGrouping(group: TIssueGroupByOptions, subgroup: TIssueGroupByOptions) {
  groupBy = group;
  subgroupBy = subgroup;
}
