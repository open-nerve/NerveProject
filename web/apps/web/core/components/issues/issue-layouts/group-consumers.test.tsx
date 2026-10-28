// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueCreate } from "@nerve/api-client";
import type { GroupByColumnTypes, TSubIssueOperations } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { labelOf, projectMemberOf, stateOf } from "@/store/project/fake-projects";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { KanBan } from "./kanban/default";
import { ListGroup } from "./list/list-group";
import { getGroupByColumns } from "./utils";
import { SubIssuesListGroup } from "../issue-detail-widgets/sub-issues/issues-list/list-group";

const bound = vi.hoisted(() => ({ app: undefined as unknown, session: { loginId: "A" } }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return bound.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: bound.session }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => ({ setToast: () => {}, TOAST_TYPE: { SUCCESS: "success", ERROR: "error" } }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => app().state }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => app().projectRoot.project }));
vi.mock("@/hooks/store/use-kanban-view", () => ({ useKanbanView: () => ({ getCanUserDragDrop: () => false }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({
  useIssueStoreType: () => EIssuesStoreType.PROJECT,
  useIssuesStore: () => ({ issues: app().issue.projectIssues }),
}));
// Row details and virtual visibility are outside the group paging/prefill boundary.
// jsdom has no layout/CSS scroll ranges; drag auto-scroll is not part of these tests.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("./kanban/blocks-list", () => ({ KanbanIssueBlocksList: () => null }));
vi.mock("./list/blocks-list", () => ({ IssueBlocksList: () => null }));
vi.mock("./kanban/headers/group-by-card", () => ({ HeaderGroupByCard: () => null }));
vi.mock("./list/headers/group-by-card", () => ({ HeaderGroupByCard: () => null }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("../create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("../issue-detail-widgets/sub-issues/issues-list/list-item", () => ({
  SubIssuesListItem: ({ issueId }: { issueId: string }) => <span>{issueId}</span>,
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const app = () => bound.app as typeof tab.app;
let container: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
let reads: Promise<unknown>[];
const intersections = new Map<Element, IntersectionObserverCallback>();
class VisibleObserver {
  constructor(private callback: IntersectionObserverCallback) {}
  observe(element: Element) {
    intersections.set(element, this.callback);
  }
  unobserve(element: Element) {
    intersections.delete(element);
  }
  disconnect() {}
}
const labelID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const memberID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb";
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal("IntersectionObserver", VisibleObserver);
  tab = await issueStoreTab();
  bound.app = tab.app;
  reads = [];
  intersections.clear();
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const noop = () => {};
const common = {
  displayProperties: undefined,
  orderBy: "sort_order" as const,
  updateIssue: undefined,
  quickActions: () => null,
  collapsedGroups: { group_by: [], sub_group_by: [] },
  handleCollapsedGroups: noop,
  canEditProperties: () => true,
  handleOnDrop: async () => {},
};
const more = (group?: string, subgroup?: string) => {
  reads.push(tab.app.issue.projectIssues.fetchNextIssues("ws", "A", group, subgroup));
};
const create = (project: string | null | undefined, body: IssueCreate) =>
  project ? tab.app.issue.projectIssues.createIssue("ws", project, body) : Promise.resolve(undefined);
const columns = (groupBy: GroupByColumnTypes | null) =>
  getGroupByColumns({ groupBy, includeNone: true, isWorkspaceLevel: false })!;

it.each([false, true])("kanban continuation remains reachable with swimlane=%s", async (lane) => {
  const row = listIssue("first", 10, { priority: "high" });
  const slice = { total: 5, data: [row], next_cursor: "server:next" };
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 1,
        query: { order_by: "sort_order", group_by: "priority", ...(lane && { sub_group_by: "label_id" as const }) },
      }),
    ["GET", "/api/v0/projects/A/issues"],
    {
      total: 5,
      groups: [lane ? { key: "high", total: 5, subgroups: [{ key: "none", ...slice }] } : { key: "high", ...slice }],
      ...(lane && { subgroup_totals: [{ key: "none", total: 5 }] }),
    },
    "kanban first page"
  );
  await act(async () =>
    view.render(
      <KanBan
        {...common}
        issuesMap={{ first: row }}
        groups={tab.app.issue.projectIssues.groups!}
        getGroupIssueCount={tab.app.issue.projectIssues.getGroupIssueCount}
        group_by="priority"
        sub_group_by={lane ? "labels" : undefined}
        {...(lane && { sub_group_id: "none" })}
        showEmptyGroup={false}
        loadMoreIssues={more}
      />
    )
  );
  if (lane) {
    const button = [...container.querySelectorAll("div")].find((node) => node.textContent === "common.load_more ↓");
    expect(button, "the swimlane exposes a continuation control").toBeDefined();
    await act(async () => button!.click());
  } else {
    expect(intersections.size, "ordinary kanban exposes an observed continuation target").toBe(1);
    await act(async () => {
      const [element, callback] = [...intersections][0]!;
      callback([{ isIntersecting: true, target: element } as IntersectionObserverEntry], {} as IntersectionObserver);
    });
  }
  await act(async () => until(() => tab.nerve.calls.length === 2, "kanban next page"));
  expect(tab.nerve.calls[1]!.query).toMatchObject({ group_key: "high", cursor: "server:next", limit: "1" });
  expect(tab.nerve.calls[1]!.query.sub_group_key).toBe(lane ? "none" : undefined);
  expect(tab.app.issue.projectIssues.getIssueLoader("high", lane ? "none" : undefined)).toBe("pagination");
  if (!lane) expect(intersections.size, "the pending part detaches its observer").toBe(0);
  await act(async () => {
    tab.nerve.calls[1]!.answer(
      json(200, { total: 5, data: [listIssue("second", 20, { priority: "high" })], next_cursor: null })
    );
    await Promise.all(reads);
  });
  expect(tab.app.issue.projectIssues.getIssueIds("high", lane ? "none" : undefined)).toEqual(["first", "second"]);
  expect(tab.app.issue.projectIssues.getGroupIssueCount("high", lane ? "none" : undefined)).toBe(5);
  expect(tab.app.issue.projectIssues.getIssueLoader("high", lane ? "none" : undefined)).toBeUndefined();
});

const cases = [
  { field: "labels", key: "none", expected: {} },
  { field: "assignees", key: "none", expected: {} },
  { field: "labels", key: labelID, expected: { label_ids: [labelID] } },
  { field: "assignees", key: memberID, expected: { assignee_ids: [memberID] } },
  { field: "priority", key: "none", expected: { priority: "none" } },
] satisfies { field: "labels" | "assignees" | "priority"; key: string; expected: Partial<IssueCreate> }[];
it.each((["list", "kanban", "swimlane"] as const).flatMap((layout) => cases.map((item) => ({ layout, item }))))(
  "$layout $item.field:$item.key production prefill reaches real quick-add HTTP",
  async ({ layout, item }) => {
    await answered(
      tab.nerve,
      () => tab.app.label.fetchProjectLabels("A"),
      ["GET", "/api/v0/projects/A/labels"],
      { data: [{ ...labelOf(tab.project, "Label", 1), id: labelID }] },
      "labels"
    );
    const member = membershipOf("Member", {}, "ws");
    await answered(
      tab.nerve,
      () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(tab.workspace),
      ["GET", "/api/v0/workspaces/ws/members"],
      { data: [{ ...member, member: { ...member.member, id: memberID } }] },
      "profiles"
    );
    await answered(
      tab.nerve,
      () => tab.app.memberRoot.project.fetchProjectMembers("A"),
      ["GET", "/api/v0/projects/A/members"],
      { data: [{ ...projectMemberOf(tab.project, "Member"), member_id: memberID }] },
      "members"
    );
    if (layout === "swimlane")
      await answered(
        tab.nerve,
        () => tab.app.state.fetchProjectStates("A"),
        ["GET", "/api/v0/projects/A/states"],
        { data: [stateOf(tab.project, "State", "unstarted", 1, { id: "state" })] },
        "states"
      );
    {
      // Each case mounts the real group producer, real prefill consumer, real RHF and root writer.
      await act(async () => view.render(null));
      const column = columns(item.field).find((group) => group.id === item.key)!;
      expect(column).toBeDefined();
      const lane = layout === "swimlane";
      const field = ({ labels: "label_id", assignees: "assignee_id", priority: "priority" } as const)[item.field];
      // Avoid grouping twice by priority in the priority:none swimlane case.
      const outer = lane ? "state_id" : field;
      const row = listIssue("existing", 10, { state_id: "state", ...item.expected });
      const slice = { total: 1, data: [row], next_cursor: null };
      await answered(
        tab.nerve,
        () =>
          tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
            limit: 10,
            query: { order_by: "sort_order", group_by: outer, ...(lane && { sub_group_by: field }) },
          }),
        ["GET", "/api/v0/projects/A/issues"],
        {
          total: 1,
          groups: [
            lane ? { key: "state", total: 1, subgroups: [{ key: item.key, ...slice }] } : { key: item.key, ...slice },
          ],
          ...(lane && { subgroup_totals: [{ key: item.key, total: 1 }] }),
        },
        "quick-add group"
      );
      const { groups } = tab.app.issue.projectIssues;
      if (layout === "list") {
        await act(async () =>
          view.render(
            <ListGroup
              {...common}
              group={column}
              group_by={item.field}
              issuesMap={{ existing: row }}
              groupIssueIds={["existing"]}
              getGroupIndex={() => 0}
              enableIssueQuickAdd
              containerRef={{ current: null }}
              quickAddCallback={create}
              showEmptyGroup
              loadMoreIssues={more}
            />
          )
        );
      } else {
        // The shared column producer supplies the swimlane's key; state columns are loaded below.
        await act(async () =>
          view.render(
            <KanBan
              {...common}
              issuesMap={{ existing: row }}
              groups={groups!}
              getGroupIssueCount={tab.app.issue.projectIssues.getGroupIssueCount}
              group_by={lane ? "state" : item.field}
              sub_group_by={lane ? item.field : undefined}
              {...(lane && { sub_group_id: column.id })}
              enableQuickIssueCreate
              quickAddCallback={create}
              showEmptyGroup={false}
              loadMoreIssues={more}
            />
          )
        );
      }
      const open = [...container.querySelectorAll("span")].find((node) => node.textContent === "issue.new");
      expect(open).toBeDefined();
      await act(async () => open!.click());
      const input = container.querySelector("input")!;
      await act(async () => {
        Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New item");
        input.dispatchEvent(new Event("input", { bubbles: true }));
      });
      const at = tab.nerve.calls.length;
      await act(async () => {
        container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
      });
      await act(async () => until(() => tab.nerve.calls.length > at, "prefill POST"));
      const call = tab.nerve.calls[at]!;
      expect([call.method, call.path]).toEqual(["POST", "/api/v0/projects/A/issues"]);
      expect(call.body).toEqual({ name: "New item", ...(lane && { state_id: "state" }), ...item.expected });
      await act(async () => {
        call.answer(json(201, { ...issueDetailOf(`new-${at}`), ...(lane && { state_id: "state" }), ...item.expected }));
        await vi.advanceTimersByTimeAsync(20);
      });
    }
  }
);

it.each([0, 24])(
  "ungrouped sub-issues at nesting offset %s render without an extra group header",
  async (spacingLeft) => {
    await act(async () =>
      view.render(
        <SubIssuesListGroup
          group={columns(null)[0]!}
          workItemIds={["child"]}
          workspaceSlug="ws"
          projectId="A"
          parentIssueId="parent"
          rootIssueId="root"
          canEdit
          spacingLeft={spacingLeft}
          handleIssueCrudState={noop}
          subIssueOperations={{} as TSubIssueOperations}
        />
      )
    );
    expect(container.textContent).toBe("child");
  }
);
