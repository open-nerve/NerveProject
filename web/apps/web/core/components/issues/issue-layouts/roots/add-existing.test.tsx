// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// "Add existing" from a planning page's empty state (P7b spec 2.9, R2, R6, R11): the empty state asks the layout's
// command host for the dialog, which searches the page's project, nerve's most at once (100), without the work items
// already in the page's cycle or module (those in another cycle are listed: adding one moves it). Each chosen one joins
// the page; once the first is confirmed the list is no longer empty and the empty state goes, while the dialog the host
// holds stays and says why nerve refused the other, which stays chosen. The host ends the dialog once the route or the
// caller's role in the project changes. The empty state offers it as it offers a new work item: to a project's admin or
// member while the page takes work items, not on a read-only page, not while filters are applied. The pages, the empty
// state, the dialog, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter, useNavigate } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { selectionButton } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf, issueOf } from "@/store/issue/fake-issues";
import { cycleRequests } from "@/store/planning/fake-cycles";
import { moduleRequests } from "@/store/planning/fake-modules";
import type { RootStore } from "@/store/root.store";
import { IssueListCommandHost } from "../command-host";
import { PlanningPageEmptyState } from "../empty-states/planning-page";
import {
  C1,
  M1,
  buttonOf,
  cyclePreferencesOf,
  layouts,
  modulePreferencesOf,
  planningPageTab,
} from "./fake-planning-page";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  navigate: undefined as ((to: string) => void) | undefined,
}));
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
// jsdom has no viewport geometry; rows, cards, days and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const C2 = "019a0000-0000-7000-8000-0000000000c2";
const BETA = "019a0000-0000-7000-8000-0000000000d2";
const GAMMA = "019a0000-0000-7000-8000-0000000000d3";
const LATER = "2026-12-31T13:00:00Z";
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  toasts.length = 0;
  bound.navigate = undefined;
});
afterEach(async () => page.close());

/** What the page's tests put before its work items: what navigates the route. */
function Navigator() {
  bound.navigate = useNavigate();
  return null;
}
const pages = {
  cycle: {
    type: EIssuesStoreType.CYCLE,
    params: { cycleId: C1 },
    preferences: cyclePreferencesOf,
    empty: "project_empty_state.cycle_work_items",
  },
  module: {
    type: EIssuesStoreType.MODULE,
    params: { moduleId: M1 },
    preferences: modulePreferencesOf,
    empty: "project_empty_state.module_work_items",
  },
} as const;
type PageName = keyof typeof pages;
/** The page with none of its work items to show, as fields and its filters say, grouped by state in its list. */
async function emptyPage(name: PageName, fields: Parameters<typeof planningPageTab>[1] = {}, filters = {}) {
  bound.params = { workspaceSlug: "ws", projectId: "A", ...pages[name].params };
  page = await planningPageTab(pages[name].type, fields);
  bound.app = page.app;
  await page.show(pages[name].preferences(layouts.list.display, filters), page.listOf("list", false), <Navigator />);
}
const addExisting = (name: PageName) => buttonOf(`${pages[name].empty}.cta_secondary`);
const text = () => document.body.textContent ?? "";
const dialog = () => document.querySelector('[role="dialog"]');
/** The work items the dialog lists, by name. */
const listed = () =>
  [...document.querySelectorAll('[role="option"]')].map((node) => node.lastElementChild?.textContent ?? "");
const rowOf = (id: string) => document.getElementById(`issue-${id}`);
const chosen = (name: string) => document.querySelector(`[aria-label="common.remove ${name}"]`) !== null;
const writes = () => page.nerve.calls.filter((call) => call.method === "PATCH");
/** A work item of the project, numbered, with fields. */
const item = (name: string, sequence: number, fields: Partial<Issue> = {}): Issue => ({
  ...issueOf(name),
  sequence_id: sequence,
  ...fields,
});

/** Opens the dialog from the empty state; nerve answers its search with the work items. */
async function searched(name: PageName, found: Issue[]) {
  await act(async () => addExisting(name)!.click());
  await until(() => page.searches().length === 1, "the dialog's search");
  await act(async () => page.searches()[0]!.answer(json(200, { data: found })));
}

it("lists the project's work items but the cycle's, joins those chosen, and keeps the dialog once the empty state goes", async () => {
  await emptyPage("cycle");
  expect(text()).toContain("project_empty_state.cycle_work_items.title");
  await searched("cycle", [item("x", 1), item("y", 2, { cycle_id: C2 }), item("z", 3, { cycle_id: C1 })]);
  expect(dialog()?.textContent).toContain("issue.add.existing");
  expect(page.searches()[0]!.query).toEqual({ project_id: "A", limit: "100" });
  expect(listed(), "the work items listed, those already in the cycle not").toEqual(["x", "y"]);
  await act(async () => selectionButton("issue.select.select_all").click());
  await act(async () => selectionButton("issue.select.add_selected").click());
  await until(() => writes().length === 2, "the writes");
  expect(writes().map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/x", { cycle_id: C1 }],
    ["/api/v0/issues/y", { cycle_id: C1 }],
  ]);
  await act(async () => {
    writes()[0]!.answer(json(200, { ...issueDetailOf("x"), sequence_id: 1, cycle_id: C1, updated_at: LATER }));
    await vi.advanceTimersByTimeAsync(100);
  });
  // the list takes x in as nerve confirmed it: its row, in place of the empty state
  expect(rowOf("x"), "x's row").not.toBeNull();
  expect(text()).not.toContain("project_empty_state.cycle_work_items.title");
  expect(dialog(), "the dialog, once the empty state is gone").not.toBeNull();
  expect([chosen("x"), chosen("y")]).toEqual([false, true]);
  await act(async () => {
    writes()[1]!.answer(problem(409, "planning.cycle_completed"));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(dialog(), "the dialog, with a refusal").not.toBeNull();
  expect(chosen("y"), "the refused one, still chosen").toBe(true);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycle_completed" }]);
  expect(page.nerve.to(cycleRequests.progress({ id: C1 }))).toHaveLength(1);
});

it("lists the project's work items but the module's, and joins one to the module, keeping its other modules", async () => {
  await emptyPage("module");
  await searched("module", [item("x", 1, { module_ids: [BETA, GAMMA] }), item("z", 2, { module_ids: [M1] })]);
  expect(page.searches()[0]!.query).toEqual({ project_id: "A", limit: "100" });
  expect(listed()).toEqual(["x"]);
  await act(async () => selectionButton("issue.select.select_all").click());
  await act(async () => selectionButton("issue.select.add_selected").click());
  await until(() => writes().length === 1, "the write");
  expect(writes()[0]!.body).toEqual({ module_ids: [BETA, GAMMA, M1] });
  await act(async () => {
    writes()[0]!.answer(json(200, { ...issueDetailOf("x"), module_ids: [BETA, GAMMA, M1], updated_at: LATER }));
    await vi.advanceTimersByTimeAsync(400);
  });
  await until(() => dialog() === null, "the dialog closed, all confirmed");
  expect(rowOf("x"), "x's row").not.toBeNull();
  expect(page.nerve.to(moduleRequests.progress({ id: M1 }))).toHaveLength(1);
});

it.each(["the route changes", "the caller's role in the project changes"])(
  "ends the dialog once %s",
  async (change) => {
    await emptyPage("cycle");
    await searched("cycle", [item("x", 1)]);
    expect(dialog()).not.toBeNull();
    if (change === "the route changes") await act(async () => bound.navigate!("/elsewhere"));
    else
      await act(async () =>
        page.app.projectRoot.project.confirmProject(page.project, (project) => ({ ...project, member_role: 20 }))
      );
    expect(dialog()).toBeNull();
  }
);

it.each([
  { who: "a member", page: "cycle", fields: {}, offered: true },
  { who: "a member", page: "module", fields: {}, offered: true },
  { who: "an admin", page: "cycle", fields: { role: 20 }, offered: true },
  { who: "a guest", page: "cycle", fields: { role: 5 }, offered: false },
  { who: "a member, the cycle ended", page: "cycle", fields: { cycle: { status: "completed" } }, offered: false },
  {
    who: "a member, the cycle archived",
    page: "cycle",
    fields: { cycle: { status: "completed", archived_at: LATER } },
    offered: false,
  },
  { who: "a member, the module archived", page: "module", fields: { module: { archived_at: LATER } }, offered: false },
] as const)("the empty state of a $page offers $who to add existing ones: $offered", async (each) => {
  await emptyPage(each.page, each.fields);
  expect(addExisting(each.page) !== undefined).toBe(each.offered);
});

it("offers no add existing in the empty state while the page's filters are applied", async () => {
  await emptyPage("cycle", {}, { priority: ["high"] });
  expect(text()).toContain("common_empty_state.search.title");
  expect(addExisting("cycle")).toBeUndefined();
});

// The page shows the cycles' turned-off state once the project's cycles are off; the empty state itself asks whether
// the page's cycle takes work items.
it("offers no add existing in the empty state while the cycle takes no work items: the project's cycles off", async () => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.render(
    <MemoryRouter>
      <IssueListCommandHost>
        <PlanningPageEmptyState storeType={EIssuesStoreType.CYCLE} />
      </IssueListCommandHost>
    </MemoryRouter>
  );
  expect(addExisting("cycle"), "while the cycles are on").toBeDefined();
  await act(async () =>
    page.app.projectRoot.project.confirmProject(page.project, (project) => ({ ...project, cycle_view: false }))
  );
  expect(addExisting("cycle"), "once they are off").toBeUndefined();
});
