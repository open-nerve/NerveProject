// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// What a project's view's page's layouts ask of the list engine by the page's store type (P6a spec 2.5, R1; re-96 I3),
// where a store type the engine missed would say nothing: the empty state's (the view page's filter, its new work item
// the view page's), a group's "+" and a row's edit and copy (dialogs the layout's command host holds, writing through
// the view page's commands; a person's page's rows keep the project page's), and a calendar day's "add existing" (the
// view page's command, as the project page's: the day alone). The layouts' parts, menus, host, stores and HTTP are
// real; the work item's dialog is not (its form has its own tests): the test reads the store type it opens with.
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { json } from "@/lib/auth/fake-nerve";
import { clickSelection, issueSelectionTab, selectionButton } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf, issueOf } from "@/store/issue/fake-issues";
import type { RootStore } from "@/store/root.store";
import { CalendarQuickAddIssueActions } from "./calendar/quick-add-issue-actions";
import { IssueListCommandHost } from "./command-host";
import { IssueLayoutEmptyState } from "./empty-states";
import { HeaderGroupByCard } from "./list/headers/group-by-card";
import { ProjectIssueQuickActions } from "./quick-action-dropdowns/project-issue";

const bound = vi.hoisted(() => ({ params: {} as Record<string, string>, opened: [] as unknown[] }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
// The calendar day's own menu is real; its quick add's form has its own tests.
vi.mock("./quick-add", () => ({
  QuickAddIssueRoot: ({ customQuickAddButton }: { customQuickAddButton: ReactNode }) => customQuickAddButton,
}));
vi.mock("@/components/issues/issue-modal/modal", () => ({
  CreateUpdateIssueModal: (props: { storeType?: EIssuesStoreType }) => {
    bound.opened.push(props.storeType);
    return null;
  },
}));

const V = "019a0000-0000-7000-8000-0000000000e1";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.params = { workspaceSlug: "ws", projectId: "A", viewId: V };
  bound.opened = [];
});
afterEach(async () => tab.close());

/** Renders children in the layouts of a page of the store type, inside the layout's command host. */
const onPage = (type: EIssuesStoreType | undefined, children: ReactNode) =>
  tab.render(
    <MemoryRouter>
      <IssuesStoreContext.Provider value={type}>
        <IssueListCommandHost>{children}</IssueListCommandHost>
      </IssuesStoreContext.Provider>
    </MemoryRouter>
  );
/** The store types the work item's dialog was opened with, each once (a dialog renders again as the host does). */
const openedWith = () => [...new Set(bound.opened)];
const buttonOf = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find((node) => node.textContent?.trim() === text);
/** The work item filters of a page: the project page's of project A, and a view page's of V, as each page made them. */
function pageFilters(viewApplies: boolean) {
  const changes = { project: vi.fn(), view: vi.fn() };
  const filters = tab.app.workItemFilters;
  filters.getOrCreateFilter({
    entityType: EIssuesStoreType.PROJECT,
    entityId: "A",
    onExpressionChange: changes.project,
  });
  const view = filters.getOrCreateFilter({
    entityType: EIssuesStoreType.PROJECT_VIEW,
    entityId: V,
    initialExpression: viewApplies ? { and: [{ priority__in: "urgent" }] } : undefined,
    onExpressionChange: changes.view,
  });
  return { changes, view };
}

it("offers the clearing of the view page's filters while any applies on a view's page with no work items", async () => {
  const { changes, view } = pageFilters(true);
  await onPage(EIssuesStoreType.PROJECT_VIEW, <IssueLayoutEmptyState storeType={EIssuesStoreType.PROJECT_VIEW} />);
  const clear = buttonOf("project_issues.empty_state.issues_empty_filter.secondary_button.text");
  expect(clear, "the clearing of the page's filters").toBeDefined();
  expect(buttonOf("project_empty_state.work_items.cta_primary")).toBeUndefined();
  await act(async () => clear!.click());
  expect(view.hasActiveFilters).toBe(false);
  expect(changes.view).toHaveBeenCalled();
  expect(changes.project).not.toHaveBeenCalled();
});

it("offers a new work item of the view's page on a view's page with no work items and no filter applied", async () => {
  pageFilters(false);
  await onPage(EIssuesStoreType.PROJECT_VIEW, <IssueLayoutEmptyState storeType={EIssuesStoreType.PROJECT_VIEW} />);
  const create = buttonOf("project_empty_state.work_items.cta_primary");
  expect(create?.disabled, "a member's new work item").toBe(false);
  await act(async () => create!.click());
  expect(tab.app.commandPalette.isCreateIssueModalOpen).toBe(true);
  expect(tab.app.commandPalette.createIssueStoreType).toBe(EIssuesStoreType.PROJECT_VIEW);
});

it("opens a group's new work item on a view's page to write through the view page's commands", async () => {
  await onPage(
    EIssuesStoreType.PROJECT_VIEW,
    <HeaderGroupByCard
      groupID="state"
      groupBy="state"
      title="Todo"
      count={1}
      issuePayload={{ state_id: "state" }}
      isExpanded
      handleCollapsedGroups={() => {}}
    />
  );
  await act(async () =>
    document.querySelector<HTMLButtonElement>('button[aria-label="create_work_item: Todo"]')!.click()
  );
  expect(openedWith()).toEqual([EIssuesStoreType.PROJECT_VIEW]);
});

it.each([
  ["a view's", EIssuesStoreType.PROJECT_VIEW, EIssuesStoreType.PROJECT_VIEW],
  ["the project's", EIssuesStoreType.PROJECT, EIssuesStoreType.PROJECT],
  ["a person's", EIssuesStoreType.PROFILE, EIssuesStoreType.PROJECT],
])("opens a row's edit and copy on %s page to write through the %s page's commands", async (_, type, writes) => {
  await onPage(
    type,
    <ProjectIssueQuickActions
      issue={issueDetailOf("x")}
      parentRef={{ current: document.createElement("div") }}
      handleDelete={async () => {}}
    />
  );
  const choose = (name: string) =>
    act(async () =>
      [...document.querySelectorAll<HTMLElement>('[data-context-menu="true"] [role="menuitem"]')]
        .find((node) => node.textContent === name)!
        .click()
    );
  await choose("common.actions.edit");
  expect(openedWith(), "the edit's dialog").toEqual([writes]);
  bound.opened = [];
  await choose("common.actions.make_a_copy");
  expect(openedWith(), "the copy's dialog").toEqual([writes]);
});

it("adds a calendar day's existing work items on a view's page through the view page's commands, the day alone", async () => {
  await onPage(
    EIssuesStoreType.PROJECT_VIEW,
    <CalendarQuickAddIssueActions prePopulatedData={{ target_date: "2026-11-01", priority: "urgent" }} />
  );
  await clickSelection('button[aria-haspopup="menu"]');
  await act(async () =>
    [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')]
      .find((element) => element.textContent === "issue.add.existing")!
      .click()
  );
  expect(tab.searches()[0]?.query, "the dialog's search of the route's project").toEqual({
    target_date: "none",
    project_id: "A",
  });
  await act(async () => tab.searches()[0]!.answer(json(200, { data: [issueOf("first")] })));
  await act(async () => selectionButton("issue.select.select_all").click());
  await act(async () => selectionButton("issue.select.add_selected").click());
  const writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes.map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/first", { target_date: "2026-11-01" }],
  ]);
  await act(async () => writes[0]!.answer(json(200, { ...issueDetailOf("first"), target_date: "2026-11-01" })));
  expect(tab.app.issue.issues.getIssueById("first")?.target_date).toBe("2026-11-01");
});
