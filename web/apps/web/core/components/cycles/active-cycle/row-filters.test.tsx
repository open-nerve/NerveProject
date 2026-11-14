// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row of the card of the cycle in progress takes its cycle's page to the row's work items (P5b spec 2.7, 2.8, R8;
// re-66 M1): a state group's (of those it shows), an assignee's, a label's, or the urgent and high ones'. The click
// replaces the page's filters with that one condition, by the whole field, read or not (one PATCH, no read), and once
// nerve confirms enters the page; a read of the page sent before it and answered older leaves it be. The urgent and high
// row also previews its work item, once the page is there, in the page's own host. A refusal is said, and the list page
// stays; a late confirmation, of a list page left or a session left, takes no one anywhere. The rows of none (no
// assignee, no labels) are plain. The stores, routes and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { toFilterExpression } from "@/lib/issue-filters";
import { listIssue, issuePageOf } from "@/store/issue/fake-issues";
import type { ProjectIssueFilters } from "@nerve/api-client";
import { ANN, BUG, openTab, partOf, rowButton, rowsIn } from "@/components/core/sidebar/fake-progress-rows";
import { C1, LIST, PAGE, answerOf, answerPreferences, arrived, cycleRoutesTab, where } from "../fake-cycle-routes";

const fixture = vi.hoisted(() => ({ app: undefined as unknown, session: { loginId: "one" } }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return fixture.app;
  },
  StoreContext: (await import("react")).createContext(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: fixture.session.loginId }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; rows and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// What the rows do not reach: the cycle's list row, the chart, the sidebar's head and details.
vi.mock("@/components/cycles/list/cycles-list-item", () => ({ CyclesListItem: () => null }));
vi.mock("@/components/core/sidebar/burndown-chart", () => ({ BurndownChart: () => null }));
vi.mock("@/components/cycles/progress-sidebar/sidebar-header", () => ({ CycleSidebarHeader: () => null }));
vi.mock("@/components/cycles/progress-sidebar/sidebar-details", () => ({ CycleSidebarDetails: () => null }));
// The preview of a work item: its host is real; what it shows, a stand-in naming the work item.
vi.mock("@/components/issues/peek-overview/view", () => ({
  IssueView: (props: { issueId: string }) => <div data-peek={props.issueId} />,
}));
vi.mock("@/components/issues/detail-host", () => ({
  IssueDetailHost: (props: { children: ReactNode }) => props.children,
}));
vi.mock("@/components/issues/issue-detail/use-issue-operations", async (original) => ({
  ...(await original<typeof import("@/components/issues/issue-detail/use-issue-operations")>()),
  useIssueOperations: () => ({ fetch: async () => undefined }),
}));

let tab: Awaited<ReturnType<typeof cycleRoutesTab>>;
beforeEach(async () => {
  localStorage.clear();
  toasts.length = 0;
  fixture.session.loginId = "one";
  tab = await cycleRoutesTab();
  fixture.app = tab.app;
});
afterEach(async () => tab.close());

const card = () => partOf("project_cycles.active_cycle.label")!;
const STARTED = "project_settings.states.groups.started";
/** The rows of the card's kinds, each by the tab it is under (none: the progress card), its name and its condition. */
const rows: { kind: string; tab?: string; name: string; filters: ProjectIssueFilters }[] = [
  { kind: "state group", name: STARTED, filters: { state_group: ["started"] } },
  {
    kind: "assignee",
    tab: "project_cycles.active_cycle.assignees",
    name: "distribution.former_member",
    filters: { assignee_id: [ANN] },
  },
  { kind: "label", tab: "project_cycles.active_cycle.labels", name: "Bug", filters: { label_id: [BUG] } },
];

/** Waits for the card's rows on the list page, and opens the tab chosen if any; gives how many requests were sent. */
async function atCard(chosen?: string) {
  await until(() => card()?.querySelector("li") != null, "the card's rows");
  if (chosen) await openTab(chosen, card());
  return tab.nerve.calls.length;
}
/** Clicks the row named name of the card, a button; the requests sent since from, once there is one. */
async function click(name: string, from: number) {
  const button = rowButton(name, card());
  expect(button, `the row ${name} is a button`).toBeDefined();
  await act(async () => button!.click());
  await until(() => tab.nerve.calls.length > from, `the request of the row ${name}`);
  return tab.nerve.calls.slice(from);
}
/** The page's list as it was last read: its query. */
const lastList = () => tab.lists().at(-1)?.query;

it.each(rows)(
  "replaces an unread cycle page's filters with a $kind row's condition by one PATCH, and enters it once confirmed",
  async ({ tab: chosen, name, filters }) => {
    await tab.open(LIST);
    const from = await atCard(chosen);
    const sent = await click(name, from);
    expect(sent[0]).toMatchObject({ method: "PATCH", path: `/api/v0/me/cycles/${C1}/preferences`, body: { filters } });
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(tab.nerve.calls.slice(from), "the one change, nothing more").toHaveLength(1);
    expect(tab.reads(), "no read of the page's preferences").toEqual([]);
    expect(where.location?.pathname, "not before nerve confirms").toBe(LIST);
    await answerPreferences(sent[0]!, filters);
    await arrived(PAGE);
    // the page reads its preferences again, after the change, and lists its work items by the row's condition
    await until(() => tab.lists().length === 1, "the page's list");
    expect(lastList()).toMatchObject({ cycle_id: C1, ...joined(filters) });
    expect(tab.app.issue.cycleIssuesFilter.getIssueFilters(C1)?.richFilters).toEqual(expressionOf(filters));
    expect(toasts).toEqual([]);
  }
);

it("replaces, never merges: a page whose filters hold other conditions gets the row's alone", async () => {
  await tab.open(PAGE);
  await until(() => tab.reads().length === 1, "the page's preferences");
  await answerPreferences(tab.reads()[0]!, { priority: ["low"], label_id: [BUG] });
  await until(() => tab.lists().length === 1, "the page's list");
  await act(async () => tab.lists()[0]!.answer(json(200, { total: 0, groups: [] })));
  await tab.go(LIST);
  const from = await atCard();
  await click(STARTED, from);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.nerve.calls.slice(from).map((call) => [call.method, call.body])).toEqual([
    ["PATCH", { filters: { state_group: ["started"] } }],
  ]);
  expect(tab.reads(), "the page's one read, before the change").toHaveLength(1);
});

it("keeps the row's filters when the page's read, sent before the change, answers older once the page is entered", async () => {
  // the page was just left, its read of its preferences still out: entered again, the page waits for that same read
  await tab.open(PAGE);
  await until(() => tab.reads().length === 1, "the page's preferences");
  await tab.go(LIST);
  const from = await atCard();
  const sent = await click(STARTED, from);
  expect(sent.map((call) => call.method)).toEqual(["PATCH"]);
  await answerPreferences(sent[0]!, { state_group: ["started"] });
  await arrived(PAGE);
  await until(() => tab.lists().length === 1, "the page's list");
  expect(tab.reads(), "the read still out is the page's").toHaveLength(1);
  await answerPreferences(tab.reads()[0]!, { priority: ["low"] });
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.app.issue.cycleIssuesFilter.getIssueFilters(C1)?.richFilters).toEqual(
    expressionOf({ state_group: ["started"] })
  );
  expect(tab.lists()).toHaveLength(1);
  expect(lastList()).toMatchObject({ state_group: "started" });
  expect(lastList()?.priority).toBeUndefined();
});

it("previews the urgent and high row's work item once the page is entered, in the page's own host", async () => {
  localStorage.setItem("activeCycleTab", JSON.stringify("Priority-Issues"));
  await tab.open(LIST);
  await until(() => tab.lists().length === 1, "the card's urgent and high work items");
  expect(tab.lists()[0]!.query).toMatchObject({ cycle_id: C1, priority: "urgent,high" });
  const urgent = listIssue("urgent", 1000, { cycle_id: C1, priority: "urgent" });
  await act(async () => tab.lists()[0]!.answer(json(200, issuePageOf([urgent]))));
  await until(() => card().querySelector('[role="tabpanel"] li') !== null, "the urgent row");
  const from = tab.nerve.calls.length;
  const sent = await click("urgent", from);
  expect(sent.map((call) => [call.method, call.body])).toEqual([
    ["PATCH", { filters: { priority: ["urgent", "high"] } }],
  ]);
  expect(tab.app.issue.issueDetail.peekIssue, "no preview on the list page").toBeUndefined();
  await answerPreferences(sent[0]!, { priority: ["urgent", "high"] });
  await arrived(PAGE);
  await until(() => document.querySelector('[data-peek="urgent"]') !== null, "the work item's preview");
  expect(tab.app.issue.issueDetail.peekIssue).toMatchObject({ workspaceSlug: "ws", projectId: "A", issueId: "urgent" });
  expect(where.location?.state ?? null, "the page took the ask: a reload does not preview again").toBeNull();
  expect(tab.lists().at(-1)?.query).toMatchObject({ cycle_id: C1, priority: "urgent,high" });
});

it("says why nerve refused the row's change, stays on the list page, and lets the row be clicked again", async () => {
  await tab.open(LIST);
  const from = await atCard();
  const sent = await click(STARTED, from);
  await act(async () => sent[0]!.answer(problem(404, "planning.cycle_not_found")));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(where.location?.pathname).toBe(LIST);
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.planning_cycle_not_found" })]);
  await click(STARTED, tab.nerve.calls.length);
  expect(tab.patches()).toHaveLength(2);
});

it("sends one change at a time: a second click while the first is out changes nothing, then or after", async () => {
  await tab.open(LIST);
  const from = await atCard();
  const sent = await click(STARTED, from);
  await act(async () => rowButton("project_settings.states.groups.completed", card())?.click());
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.nerve.calls.slice(from)).toEqual(sent);
  await answerPreferences(sent[0]!, { state_group: ["started"] });
  await arrived(PAGE);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tab.patches()).toHaveLength(1);
  expect(tab.app.issue.cycleIssuesFilter.getIssueFilters(C1)?.richFilters).toEqual(
    expressionOf({ state_group: ["started"] })
  );
});

for (const ending of ["the list page left", "another session"] as const)
  it.each(["confirms", "refuses"] as const)(
    `takes no one anywhere, says nothing, when nerve %s after ${ending}`,
    async (outcome) => {
      await tab.open(LIST);
      const from = await atCard();
      const sent = await click(STARTED, from);
      if (ending === "another session") fixture.session.loginId = "two";
      else await tab.go("/ws/projects/A/issues");
      const at = where.location?.pathname;
      await act(async () =>
        sent[0]!.answer(
          outcome === "confirms"
            ? json(200, answerOf({ state_group: ["started"] }))
            : problem(404, "planning.cycle_not_found")
        )
      );
      await act(async () => vi.advanceTimersByTimeAsync(100));
      expect(where.location?.pathname).toBe(at);
      expect(toasts).toEqual([]);
    }
  );

it("makes the rows of a value buttons named by the row, and leaves the rows of none plain", async () => {
  await tab.open(LIST);
  await atCard("project_cycles.active_cycle.assignees");
  const panel = () => card().querySelector('[role="tabpanel"]:not([aria-hidden])')!;
  expect(rowsIn(panel())).toEqual([
    [true, "distribution.former_member"],
    [false, "no_assignee"],
  ]);
  await openTab("project_cycles.active_cycle.labels", card());
  expect(rowsIn(panel())).toEqual([
    [true, "Bug"],
    [true, "UI"],
    [false, "distribution.no_labels"],
  ]);
  const bug = rowButton("Bug", card())!;
  expect(bug.type).toBe("button");
  expect(bug.hasAttribute("aria-pressed"), "a row of the card sets the filters, it does not toggle").toBe(false);
  // the progress card's rows, of the state groups that have work items
  const progress = partOf("project_cycles.active_cycle.progress")!;
  expect(rowsIn(progress)).toEqual([
    [true, "project_settings.states.groups.completed"],
    [true, STARTED],
    [true, "project_settings.states.groups.backlog"],
  ]);
});

/** The filters as the list's query sends them: each field's values joined. */
const joined = (filters: ProjectIssueFilters) =>
  Object.fromEntries(Object.entries(filters).map(([field, values]) => [field, values.join(",")]));
/** The filters as the page's conditions show them. */
const expressionOf = (filters: ProjectIssueFilters) => toFilterExpression(filters);
