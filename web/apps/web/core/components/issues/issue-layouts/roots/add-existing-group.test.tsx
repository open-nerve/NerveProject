// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// "Add existing" from a group's header (P7b spec 2.9, R5, R11, R20): on a cycle's or a module's page, beside the "+" of a
// list group's or a board column's header, a second button named by the group asks the layout's command host for the
// dialog; the work items chosen join the page, not the group (nothing of the group's value is sent). The host holds the
// dialog while the list is checked, the headers gone. It is offered where the "+" is, while the page's cycle or module
// takes work items: not on the project's page, not to a guest, not on a read-only page, not on a column nerve refuses
// (an archived module's), not once the project's cycles are off. The pages, headers, dialog, stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { selectionButton } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf, issueOf, listIssue } from "@/store/issue/fake-issues";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf, preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { IssueListCommandHost } from "../command-host";
import { HeaderGroupByCard } from "../list/headers/group-by-card";
import {
  C1,
  M1,
  answeringPlanningLists,
  cyclePreferencesOf,
  layouts,
  modulePreferencesOf,
  planningPageTab,
  rowShown,
  type Layout,
} from "./fake-planning-page";
import { ProjectLayoutRoot } from "./project-layout-root";

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
// jsdom has no viewport geometry; rows, cards and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const ARCHIVED = "2026-10-20T09:00:00Z";
let page: Awaited<ReturnType<typeof planningPageTab>>;
afterEach(async () => page.close());

/** The groups' add existing, by their names. */
const entries = () =>
  [...document.querySelectorAll('button[aria-label^="issue.add.existing: "]')].map((node) =>
    node.getAttribute("aria-label")
  );
const entry = (group: string) =>
  document.querySelector<HTMLButtonElement>(`button[aria-label="issue.add.existing: ${group}"]`);
/** The group's header, by its toggle, named by the group. */
const groupShown = (group: string) => document.querySelector(`button[aria-label="${group}"][aria-expanded]`) !== null;
const options = () => [...document.querySelectorAll('[role="option"]')].map((node) => node.textContent);
const writes = () => page.nerve.calls.filter((call) => call.method === "PATCH");
/** A work item of the project, numbered, of state Todo. */
const item = (name: string, sequence: number, fields: Partial<Issue> = {}): Issue => ({
  ...issueOf(name),
  sequence_id: sequence,
  ...fields,
});

/** Each planning page a group's add existing is on: its route, its layout, and what joining it sends. */
const pages: Record<
  string,
  { type: EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE; params: object; layout: Layout; joins: object }
> = {
  "a cycle's list": { type: EIssuesStoreType.CYCLE, params: { cycleId: C1 }, layout: "list", joins: { cycle_id: C1 } },
  "a cycle's board": {
    type: EIssuesStoreType.CYCLE,
    params: { cycleId: C1 },
    layout: "kanban",
    joins: { cycle_id: C1 },
  },
  "a module's list": {
    type: EIssuesStoreType.MODULE,
    params: { moduleId: M1 },
    layout: "list",
    joins: { module_ids: [M1] },
  },
};
/** Shows the page in its layout grouped by state, its one work item in the group Todo, the caller's fields as given. */
async function shown(name: string, fields: Parameters<typeof planningPageTab>[1] = {}) {
  const { type, params, layout } = pages[name]!;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...params };
  page = await planningPageTab(type, fields);
  bound.app = page.app;
  const preferences = type === EIssuesStoreType.CYCLE ? cyclePreferencesOf : modulePreferencesOf;
  await page.show(preferences(layouts[layout].display), page.listOf(layout));
  await until(() => groupShown("Todo"), "the group Todo");
}
/** Opens the dialog from the group's add existing; nerve answers its search with the work items. */
async function searched(group: string, found: Issue[]) {
  await act(async () => entry(group)!.click());
  await until(() => page.searches().length === 1, "the dialog's search");
  await act(async () => page.searches()[0]!.answer(json(200, { data: found })));
}
/** Chooses each work item listed and adds them. */
async function addAll() {
  await act(async () => selectionButton("issue.select.select_all").click());
  await act(async () => selectionButton("issue.select.add_selected").click());
}

it.each(Object.keys(pages))(
  "offers on %s an add existing beside its group's +, named by the group, that joins the chosen ones to the page only",
  async (name) => {
    await shown(name);
    expect(entries()).toEqual(["issue.add.existing: Todo"]);
    expect(document.querySelector('button[aria-label="create_work_item: Todo"]'), "the group's +").not.toBeNull();
    await searched("Todo", [item("x", 2)]);
    expect(page.searches()[0]!.query).toEqual({ project_id: "A", limit: "100" });
    expect(options()).toEqual(["A-2x"]);
    await addAll();
    await until(() => writes().length === 1, "the write");
    expect(writes()[0]!.body, "the page's own, not the group's state").toEqual(pages[name]!.joins);
  }
);

it("keeps the dialog a group's add existing opens while the list is checked, the groups' headers gone", async () => {
  await shown("a cycle's list");
  await searched("Todo", [item("x", 2)]);
  // the work item is checked against nerve (as its preview's read does): the list shows its loader meanwhile
  void page.app.issue.issues.refreshAuthoritative("first").catch(() => undefined);
  const checks = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "GET");
  await until(() => checks().length === 1, "the check of the work item");
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect([rowShown(), entry("Todo")], "the rows and the groups' headers, while the list is checked").toEqual([
    false,
    null,
  ]);
  expect(options(), "the dialog, while the list is checked").toEqual(["A-2x"]);
  const [row] = (layouts.spreadsheet.page() as { data: IssueDetail[] }).data;
  await act(async () => checks()[0]!.answer(json(200, { ...issueDetailOf("first"), ...row })));
  await until(() => page.lists().length === 2, "the list's read again");
  await act(async () => page.lists()[1]!.answer(json(200, page.listOf("list"))));
  await until(rowShown, "the row again");
  await addAll();
  await until(() => writes().length === 1, "the write");
  expect(writes()[0]!.body).toEqual({ cycle_id: C1 });
});

it.each([
  { who: "a guest", name: "a cycle's list", fields: { role: 5 } },
  { who: "a member, the cycle ended", name: "a cycle's board", fields: { cycle: { status: "completed" } } },
  {
    who: "a member, the cycle archived",
    name: "a cycle's list",
    fields: { cycle: { status: "completed", archived_at: ARCHIVED } },
  },
  { who: "a member, the module archived", name: "a module's list", fields: { module: { archived_at: ARCHIVED } } },
] as const)("offers no add existing in the groups of $name to $who", async ({ name, fields }) => {
  await shown(name, fields);
  expect(entries()).toEqual([]);
});

it("offers no add existing in the groups of the project's page, beside their +", async () => {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const preferences = () => page.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the project's preferences");
  const display = { ...preferencesOf().display_filters, ...layouts.list.display };
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf({ display_filters: display }))));
  await until(() => page.lists().length === 1, "the project's list");
  await act(async () => page.lists()[0]!.answer(json(200, layouts.list.page())));
  await until(() => groupShown("Todo"), "the group Todo");
  expect(document.querySelector('button[aria-label="create_work_item: Todo"]'), "the group's +").not.toBeNull();
  expect(entries()).toEqual([]);
});

it.each(["list", "kanban"] as const)(
  "offers no add existing on a column nerve refuses, an archived module's, of a cycle's %s grouped by module",
  async (layout) => {
    bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
    page = await planningPageTab(EIssuesStoreType.CYCLE);
    bound.app = page.app;
    const alpha = moduleOf(page.project, "Alpha");
    const gamma = moduleOf(page.project, "Gamma", { archived_at: ARCHIVED });
    answeringPlanningLists({
      cycles: [rowOf(cycleOf(page.project, "Now", { id: C1, status: "current" }))],
      modules: [moduleRowOf(alpha)],
      archivedModules: [moduleRowOf(gamma)],
    });
    const group = (key: string, modules: string[]) => ({
      key,
      total: 1,
      data: [listIssue(`in-${key}`, 1000, { cycle_id: C1, module_ids: modules })],
      next_cursor: null,
    });
    await page.show(cyclePreferencesOf({ layout, group_by: "module_id", order_by: "sort_order" }), {
      total: 3,
      groups: [group(alpha.id, [alpha.id]), group(gamma.id, [gamma.id]), group("none", [])],
    });
    await until(() => groupShown("Gamma, common.archived"), "the archived module's group, by its name");
    expect(entries()).toEqual(["issue.add.existing: Alpha", "issue.add.existing: None"]);
  }
);

// The page shows the cycles' turned-off state once the project's cycles are off; the header itself asks whether the
// page's cycle takes work items.
it("offers no add existing in a group's header of a cycle's page once the cycle takes no work items: the project's cycles off", async () => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  await page.render(
    <MemoryRouter>
      <IssuesStoreContext.Provider value={EIssuesStoreType.CYCLE}>
        <IssueListCommandHost>
          <HeaderGroupByCard
            groupID="state"
            groupBy="state"
            title="Todo"
            count={1}
            issuePayload={{ state_id: "state" }}
            isExpanded
            handleCollapsedGroups={() => {}}
          />
        </IssueListCommandHost>
      </IssuesStoreContext.Provider>
    </MemoryRouter>
  );
  expect(entry("Todo"), "while the cycles are on").not.toBeNull();
  await act(async () =>
    page.app.projectRoot.project.confirmProject(page.project, (project) => ({ ...project, cycle_view: false }))
  );
  expect(entry("Todo"), "once they are off").toBeNull();
});
