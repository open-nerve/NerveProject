// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The card of the cycle in progress's tab of its priority work items (P5b spec 2.7, R7; re-66 M4): the card's tabs in
// the spec's order (priority work items, assignees, labels), the assignees' still the one first shown and the one
// chosen kept; once open, the tab reads the cycle's urgent and high work items by their own whole query, of the
// project's members alone (again once the caller's role changes), of a project not archived whose cycles are on. A row
// says the work item's identifier, name, priority, state and target date; without one, the tab says how to see one;
// until nerve answers, it waits, and says why nerve could not, with a retry. Scrolled to its end, it reads the next
// page by its cursor. A card mounted again makes no second list, and a late answer of a session left writes nothing
// while the new session reads its own. The stores and HTTP are real.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, Project } from "@nerve/api-client";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { StoreContext } from "@/lib/store-context";
import { issueOf, issuePageOf, issueStoreTab } from "@/store/issue/fake-issues";
import { progressOf } from "@/store/planning/fake-cycles";
import { cycleOf, stateOf } from "@/store/project/fake-projects";
import { ActiveCycleStats } from "./cycle-stats";

const fixture = vi.hoisted(() => ({ session: { loginId: "one" } }));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return { StoreContext: createContext(undefined), rootStore: undefined };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: fixture.session.loginId }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const C1 = "019a0000-0000-7000-8000-0000000000c1";
const PRIORITY = "project_cycles.active_cycle.priority_issue";
const ASSIGNEES = "project_cycles.active_cycle.assignees";
const LABELS = "project_cycles.active_cycle.labels";
type Tab = Awaited<ReturnType<typeof issueStoreTab>>;
let tab: Tab;
const opened: Tab[] = [];
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
/** One SWR cache for the test, as the app keeps one across its pages. */
let cache: Map<string, unknown>;
/** The scroll sentinels observed, by the callback their observer calls. */
const intersections = new Map<Element, IntersectionObserverCallback>();

/** A tab of the session's stores: project A, of the caller's role and its fields, its one state "Doing". */
async function openTab(project: Partial<Project> = {}) {
  const one = await issueStoreTab();
  opened.push(one);
  one.nerve.respond = (call) => {
    if (call.path === "/api/v0/projects/A/states")
      call.answer(json(200, { data: [stateOf(one.project, "Doing", "started", 1, { id: "state" })] }));
  };
  await one.app.state.fetchProjectStates("A");
  one.app.projectRoot.project.confirmProject(one.project, (held) => ({ ...held, ...project }));
  one.nerve.calls.length = 0;
  return one;
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      constructor(private readonly callback: IntersectionObserverCallback) {}
      observe(target: Element) {
        intersections.set(target, this.callback);
      }
      unobserve(target: Element) {
        intersections.delete(target);
      }
      disconnect() {}
    }
  );
  localStorage.clear();
  intersections.clear();
  cache = new Map();
  fixture.session.loginId = "one";
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  host.remove();
  for (const one of opened.splice(0)) one.app.dispose();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const progress = () => progressOf(cycleOf(tab.project, "Now", { id: C1, status: "current" })).live;
/** Renders the card's breakdowns of cycle C1 in the stores of app, as many cards as copies. */
async function render(app = tab.app, copies = 1) {
  await act(async () =>
    view.render(
      <StoreContext.Provider value={app as never}>
        <SWRConfig value={{ provider: () => cache as never }}>
          {Array.from({ length: copies }, (_, at) => (
            <ActiveCycleStats
              key={at}
              workspaceSlug="ws"
              projectId="A"
              cycleId={C1}
              progress={progress()}
              enterFiltered={() => undefined}
            />
          ))}
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
}
/** Renders the card with the tab of priority work items chosen before. */
async function renderOpen(app = tab.app, copies = 1) {
  localStorage.setItem("activeCycleTab", JSON.stringify("Priority-Issues"));
  await render(app, copies);
}
const lists = (one = tab) => one.nerve.to("/api/v0/projects/A/issues");
const tabs = () => [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')];
const tabNamed = (name: string) => tabs().find((one) => one.textContent === name);
const panel = () => host.querySelector<HTMLElement>('[role="tabpanel"]');
/** The rows of the open tab, each as it reads. */
const rows = () => [...host.querySelectorAll('[role="tabpanel"] li')].map((row) => row.textContent);
const button = (text: string) => [...host.querySelectorAll("button")].find((one) => one.textContent === text);
/** Answers the first read of the cycle's priority work items. */
async function answerFirst(rowsOf: Issue[], next: string | null = null, one = tab) {
  await until(() => lists(one).length > 0, "the tab's read");
  await act(async () =>
    lists(one)[0]!.answer(json(200, { ...issuePageOf(rowsOf, next), total: rowsOf.length + (next ? 1 : 0) }))
  );
}
/** Scrolls the open tab to its end: its sentinel comes into view. */
const scrollToEnd = () =>
  act(async () => {
    for (const [target, callback] of intersections)
      callback([{ isIntersecting: true, target } as IntersectionObserverEntry], {} as IntersectionObserver);
  });

const urgent = (id: string, sequence: number, created_at: string, fields: Partial<Issue> = {}): Issue => ({
  ...issueOf(id),
  name: `Fix ${id}`,
  sequence_id: sequence,
  priority: "urgent",
  cycle_id: C1,
  created_at,
  ...fields,
});
const login = urgent("login", 7, "2026-11-03T00:00:00Z", { target_date: "2026-11-15" });
const cache2 = urgent("cache", 4, "2026-11-01T00:00:00Z", { priority: "high" });
const LOGIN = "A-7Fix loginpriority: urgentDoingNov 15";
const CACHE = "A-4Fix cachepriority: highDoing";

it("has its tabs in their order, the assignees' first, and reads once the priority work items' is open", async () => {
  tab = await openTab();
  await render();
  expect(tabs().map((one) => one.textContent)).toEqual([PRIORITY, ASSIGNEES, LABELS]);
  expect(tabNamed(ASSIGNEES)?.getAttribute("aria-selected")).toBe("true");
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(lists()).toEqual([]);
  await act(async () => tabNamed(PRIORITY)?.click());
  expect(tabNamed(PRIORITY)?.getAttribute("aria-selected")).toBe("true");
  expect(localStorage.getItem("activeCycleTab")).toBe('"Priority-Issues"');
  await until(() => lists().length === 1, "the tab's read");
  expect(lists()[0]!.query).toEqual({
    cycle_id: C1,
    priority: "urgent,high",
    order_by: "-created_at",
    sub_issue: "true",
    limit: "30",
  });
});

it("waits for nerve's answer, then shows each work item by identifier, name, priority, state and target date", async () => {
  tab = await openTab();
  await renderOpen();
  expect(panel()?.querySelector('[role="status"]')).not.toBeNull();
  expect(rows()).toEqual([]);
  await answerFirst([login, cache2]);
  expect(panel()?.querySelector('[role="status"]')).toBeNull();
  expect(rows()).toEqual([LOGIN, CACHE]);
});

it("says how to see one when the cycle has no urgent or high work item", async () => {
  tab = await openTab();
  await renderOpen();
  await answerFirst([]);
  expect(rows()).toEqual([]);
  expect(panel()?.textContent).toContain("active_cycle.empty_state.priority_issue.title");
  expect(panel()?.querySelector('[role="status"]')).toBeNull();
});

it("says why nerve could not read them, and reads them again on retry", async () => {
  tab = await openTab();
  await renderOpen();
  await until(() => lists().length === 1, "the tab's read");
  await act(async () => lists()[0]!.answer(problem(503, "server_busy")));
  expect(panel()?.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  await act(async () => button("common.retry")?.click());
  await until(() => lists().length === 2, "the read again");
  await act(async () => lists()[1]!.answer(json(200, issuePageOf([login]))));
  expect(panel()?.querySelector('[role="alert"]')).toBeNull();
  expect(rows()).toEqual([LOGIN]);
});

it("reads the next page by its cursor once scrolled to its end, after the rows it has", async () => {
  tab = await openTab();
  await renderOpen();
  await answerFirst([login], "next-1");
  expect(rows()).toEqual([LOGIN]);
  expect(intersections.size).toBe(1);
  await scrollToEnd();
  await until(() => lists().length === 2, "the next page");
  expect(lists()[1]!.query).toEqual({ ...lists()[0]!.query, cursor: "next-1" });
  // a failed page says why, with a retry of it, and no longer reads by itself
  await act(async () => lists()[1]!.answer(problem(503, "server_busy")));
  expect(panel()?.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(intersections.size).toBe(0);
  expect(rows()).toEqual([LOGIN]);
  await act(async () => button("common.retry")?.click());
  await until(() => lists().length === 3, "the next page again");
  expect(lists()[2]!.query).toEqual(lists()[1]!.query);
  await act(async () => lists()[2]!.answer(json(200, { ...issuePageOf([cache2]), total: 2 })));
  expect(rows()).toEqual([LOGIN, CACHE]);
  expect(panel()?.querySelector('[role="alert"]')).toBeNull();
  expect(intersections.size).toBe(0);
  await scrollToEnd();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(lists()).toHaveLength(3);
});

it.each([
  ["a caller who is not the project's member", { member_role: null }],
  ["an archived project", { archived_at: "2026-11-01T00:00:00Z" }],
  ["a project whose cycles are off", { cycle_view: false }],
] as const)("reads nothing, and makes no list, for %s", async (_, project) => {
  tab = await openTab(project);
  const registered = vi.spyOn(tab.app.issue, "registerList");
  await renderOpen();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(tabNamed(PRIORITY)?.getAttribute("aria-selected"), "the tab, open").toBe("true");
  expect(panel(), "its panel").not.toBeNull();
  expect(lists()).toEqual([]);
  expect(registered).not.toHaveBeenCalled();
});

it("reads them for a guest of the project too", async () => {
  tab = await openTab({ member_role: 5 });
  await renderOpen();
  await answerFirst([login]);
  expect(rows()).toEqual([LOGIN]);
});

it("reads them again once the caller's role in the project changes, as a guest sees his own alone", async () => {
  tab = await openTab();
  await renderOpen();
  await answerFirst([login, cache2]);
  expect(rows()).toEqual([LOGIN, CACHE]);
  await act(async () =>
    tab.app.projectRoot.project.confirmProject(tab.project, (held) => ({ ...held, member_role: 5 }))
  );
  await until(() => lists().length === 2, "the read under the new role");
  await act(async () => lists()[1]!.answer(json(200, issuePageOf([cache2]))));
  expect(rows()).toEqual([CACHE]);
});

it("makes the cycle's one list, however many cards show it and however often they are mounted", async () => {
  tab = await openTab();
  const registered = vi.spyOn(tab.app.issue, "registerList");
  await renderOpen(tab.app, 2);
  await answerFirst([login]);
  expect(rows()).toEqual([LOGIN, LOGIN]);
  await act(async () => view.unmount());
  view = createRoot(host);
  await renderOpen();
  await until(() => lists().length === 2, "the read as the card mounts again");
  await act(async () => lists()[1]!.answer(json(200, issuePageOf([cache2]))));
  expect(rows()).toEqual([CACHE]);
  expect(registered).toHaveBeenCalledTimes(1);
});

it("writes nothing of a late answer once the session has changed, and reads the new session's own", async () => {
  tab = await openTab();
  await renderOpen();
  await until(() => lists().length === 1, "the first session's read");
  const late = lists()[0]!;
  // another session starts: its stores take over, the first's retire (store-context.tsx)
  const next = await openTab();
  fixture.session.loginId = "two";
  tab.app.dispose();
  await render(next.app);
  await act(async () => late.answer(json(200, issuePageOf([login]))));
  expect(tab.app.issue.issues.held("login")).toBeUndefined();
  expect(tab.app.issue.cyclePriorityIssues.view("ws", { projectID: "A", cycleID: C1 })).toBeUndefined();
  expect(rows()).toEqual([]);
  await answerFirst([cache2], null, next);
  expect(rows()).toEqual([CACHE]);
});
