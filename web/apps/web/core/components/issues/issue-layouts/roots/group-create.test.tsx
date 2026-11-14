// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A group's new work item (M6 P7a spec 2.7, R16; re-72 F4): the "+" of a list group's or a kanban column's header asks
// the layout's command host for the new work item's dialog, with what the group gives it and the page's store type.
// While the list is checked the layout shows its loader, the groups and their headers gone, and the dialog stays open
// with what was typed in it; it sends one request. The host holds it by the route's project; on a page of no project,
// by the group's project when grouped by project, else by the caller's workspace. It keeps a cycle's page's project,
// not the project's page's. The pages, layouts, headers, the dialog's form, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueDetail } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { IssueListCommandHost } from "../command-host";
import { HeaderGroupByCard } from "../list/headers/group-by-card";
import {
  C1,
  creates,
  cyclePreferencesOf,
  form,
  layouts,
  planningPageTab,
  projectField,
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
// jsdom has no viewport geometry, dialog transitions, editor or font loading; the form's own fields are real, its
// default properties have their own tests (planning-create.test.tsx).
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ...(await import("@/components/issues/issue-modal/fake-dialog")).modal,
}));
vi.mock(
  "@/components/editor/rich-text",
  async () => (await import("@/components/issues/issue-modal/fake-dialog")).editor
);
vi.mock("@/components/issues/issue-modal/components/default-properties", () => ({
  IssueDefaultProperties: () => null,
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));

let page: Awaited<ReturnType<typeof planningPageTab>>;
afterEach(async () => page.close());
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
});

/** The "+" of the first group's header, which opens its new work item. */
const plus = () => document.querySelector<HTMLButtonElement>('button[aria-label^="create_work_item: "]');
/** The page's work item "first" as nerve reads it: the row's own fields, with its description. */
const first = (): IssueDetail => {
  const [row] = (layouts.spreadsheet.page() as { data: IssueDetail[] }).data;
  return { ...issueDetailOf("first"), ...row };
};

/** Each page a group's + is on: its route, how the test shows it in a layout, and whether its form keeps the project. */
const pages: Record<string, { params: Record<string, string>; layout: Layout; locked: boolean; cycle?: boolean }> = {
  "the project's list": { params: {}, layout: "list", locked: false },
  "the project's kanban": { params: {}, layout: "kanban", locked: false },
  "a cycle's list": { params: { cycleId: C1 }, layout: "list", locked: true, cycle: true },
};
/** Shows the page in its layout, its one work item answered, grouped by state. */
async function shown(name: string) {
  const { params, layout, cycle } = pages[name]!;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...params };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  const list = layouts[layout].page();
  if (cycle) return page.show(cyclePreferencesOf(layouts[layout].display), list);
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const preferences = () => page.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the project's preferences");
  const display = { ...preferencesOf().display_filters, ...layouts[layout].display };
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf({ display_filters: display }))));
  await until(() => page.lists().length === 1, "the project's list");
  await act(async () => page.lists()[0]!.answer(json(200, list)));
}
/** Types name as the new work item's in the open dialog. */
const type = (name: string) =>
  act(async () => {
    const input = form()!.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, name);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
const typed = () => form()?.querySelector<HTMLInputElement>("#name")?.value;

it.each(Object.keys(pages))(
  "keeps the new work item's dialog a state group's + opens on %s open while the list is checked, and sends it once",
  async (name) => {
    await shown(name);
    await until(rowShown, "the row");
    await act(async () => plus()!.click());
    expect(form(), "the dialog").not.toBeNull();
    expect(projectField()?.disabled, "whether the form keeps the page's project").toBe(pages[name]!.locked);
    await type("Kept");
    // the work item is checked against nerve (as its preview's read does): the list shows its loader meanwhile
    void page.app.issue.issues.refreshAuthoritative("first").catch(() => undefined);
    const checks = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "GET");
    await until(() => checks().length === 1, "the check of the work item");
    await act(async () => vi.advanceTimersByTimeAsync(20));
    expect([rowShown(), plus()], "the rows and the groups' headers, while the list is checked").toEqual([false, null]);
    expect(typed(), "the dialog, while the list is checked").toBe("Kept");
    await act(async () => checks()[0]!.answer(json(200, first())));
    await until(() => page.lists().length === 2, "the list's read again");
    await act(async () => page.lists()[1]!.answer(json(200, layouts[pages[name]!.layout].page())));
    await until(rowShown, "the row again");
    expect(typed(), "the dialog, once the list is back").toBe("Kept");
    await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    await until(() => creates().length === 1, "the creation");
    expect(creates()[0]!.body).toMatchObject({ name: "Kept", state_id: "state" });
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(creates()).toHaveLength(1);
  }
);

/**
 * On a page of no project (a person's work items), a group's + as the header of a group of its list gives it, with
 * payload, in the layout's command host; projects A and B are the caller's, a member's of both.
 */
async function personsGroup(payload: Partial<IssueDetail>) {
  bound.params = { workspaceSlug: "ws", userId: "u-actor" };
  page = await planningPageTab(EIssuesStoreType.CYCLE);
  bound.app = page.app;
  const other = projectOf("B", page.workspace.id, { id: "B" });
  await loadProjects(page.nerve, page.app.projectRoot.project, page.workspace, [page.project, other]);
  await page.render(
    <MemoryRouter>
      <IssuesStoreContext.Provider value={EIssuesStoreType.PROFILE}>
        <IssueListCommandHost>
          <HeaderGroupByCard
            groupID="group"
            groupBy="project"
            title="Group"
            count={1}
            issuePayload={payload}
            isExpanded
            handleCollapsedGroups={() => {}}
          />
        </IssueListCommandHost>
      </IssuesStoreContext.Provider>
    </MemoryRouter>
  );
  await act(async () => plus()!.click());
  return other;
}
/** The caller's role in the project changes, as nerve confirms it. */
const roleChanges = (project: Parameters<RootStore["projectRoot"]["project"]["confirmProject"]>[0]) =>
  act(async () =>
    page.app.projectRoot.project.confirmProject(project, (held) => ({ ...held, member_role: 20 as const }))
  );

it("holds the dialog of a project's group's + on a page of no project by the group's project: it ends as the caller's role there changes, not elsewhere", async () => {
  const other = await personsGroup({ project_id: "A" });
  expect(projectField()?.disabled, "the form lets the project be changed").toBe(false);
  await roleChanges(other);
  expect(form(), "the dialog, the caller's role in another project changed").not.toBeNull();
  await roleChanges(page.project);
  expect(form(), "the dialog, the caller's role in the group's project changed").toBeNull();
});

it("holds the dialog of another group's + on a page of no project by the caller's workspace", async () => {
  await personsGroup({ priority: "high" });
  expect(form(), "the dialog").not.toBeNull();
  await roleChanges(page.project);
  expect(form(), "the dialog, the caller's role in a project of the workspace changed").toBeNull();
});
