// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Save view makes a new view of what a list page shows (P6a spec 2.4, 2.7, R7, R8, R30): the project's page, a cycle's,
// a module's and a project's view's page each offer it; its dialog starts with no name; the new view keeps the page's
// conditions, a cycle's or a module's page's own term besides them (its cycle, its module), and the display the page's
// preferences confirmed, in nerve's words (its draft's, on a view's page), never the controls' words. The pages, the
// stores, the dialog and HTTP are real; the work item's preview is left out.
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import type { IssueListDisplayFilters, ProjectPreferences } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ProjectLayoutRoot } from "@/components/issues/issue-layouts/roots/project-layout-root";
import {
  C1,
  cyclePreferencesOf,
  layouts,
  M1,
  modulePreferencesOf,
  planningPageTab,
} from "@/components/issues/issue-layouts/roots/fake-planning-page";
import { settle } from "@/components/common/fake-quick-actions";
import { field, fill, submit } from "@/components/views/fake-view-dialog";
import { type Call, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { buttonOf, viewSavingTab } from "./fake-view-saving";

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
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => <span>the project</span> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => <header>{props.header}</header>,
}));

/** The page's conditions: its work items of high priority. */
const CONDITIONS = { priority: ["high"] };
/** A page's display in nerve's words: a list grouped by state, by its name (the controls': state, state__name). */
const GROUPED: Partial<IssueListDisplayFilters> = { layout: "list", group_by: "state_id", order_by: "state_name" };
/** The project A as an admin holds it, its views on. */
const ADMIN = { role: 20, project: { issue_views_view: true } } as const;

let close: (() => Promise<void>) | undefined;
let creations: () => Call[];
afterEach(async () => {
  await close?.();
  close = undefined;
});

/** Saves the page's view as "Mine" from its filters row: the dialog starts with no name; gives what was sent. */
async function saveAs() {
  await settle();
  const save = buttonOf("common.filter_options.save_view");
  expect(save, "Save view").toBeDefined();
  await act(async () => save!.click());
  await settle();
  expect(field("name").value, "a new view's name starts empty").toBe("");
  await fill("name", "Mine");
  await submit();
  await until(() => creations().length === 1, "the new view");
  return creations()[0]!.body;
}

it("saves the project's page as a view of its conditions and of the display its preferences hold", async () => {
  const tab = await issueSelectionTab();
  bound.app = tab.app;
  close = tab.close;
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  creations = () => tab.nerve.to("/api/v0/projects/A/views").filter((call) => call.method === "POST");
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({
    ...project,
    member_role: ADMIN.role,
    ...ADMIN.project,
  }));
  await tab.app.state.fetchProjectStates("A");
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const read: ProjectPreferences = preferencesOf({ filters: CONDITIONS });
  read.display_filters = { ...read.display_filters, ...GROUPED };
  const preferences = () => tab.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the page's preferences");
  await act(async () => preferences()[0]!.answer(json(200, read)));
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  await until(() => lists().length === 1, "the page's list");
  await act(async () => lists()[0]!.answer(json(200, layouts.list.page())));
  expect(await saveAs()).toEqual({
    name: "Mine",
    description: "",
    logo_props: {},
    filters: CONDITIONS,
    display_filters: read.display_filters,
    display_properties: read.display_properties,
  });
});

it.each([
  ["a cycle's", EIssuesStoreType.CYCLE, { cycle_id: [C1] }],
  ["a module's", EIssuesStoreType.MODULE, { module_id: [M1] }],
] as const)("saves %s page as a view of its conditions and its own term, with its display", async (_, type, term) => {
  const page = await planningPageTab(type, ADMIN);
  bound.app = page.app;
  close = page.close;
  const cycle = type === EIssuesStoreType.CYCLE;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...(cycle ? { cycleId: C1 } : { moduleId: M1 }) };
  creations = () => page.nerve.to("/api/v0/projects/A/views").filter((call) => call.method === "POST");
  const preferences = (cycle ? cyclePreferencesOf : modulePreferencesOf)(GROUPED, CONDITIONS);
  await page.show(preferences, page.listOf("list"));
  const body = await saveAs();
  expect(body).toEqual({
    name: "Mine",
    description: "",
    logo_props: {},
    filters: { ...CONDITIONS, ...term },
    display_filters: preferences.display_filters,
    display_properties: preferences.display_properties,
  });
});

it("saves a view's page as a new view of its draft: its conditions and its display", async () => {
  const page = await viewSavingTab({
    role: 20,
    view: { display_filters: { ...preferencesOf().display_filters, ...GROUPED } },
  });
  bound.app = page.app;
  close = page.close;
  bound.params = { workspaceSlug: "ws", projectId: "A", viewId: page.view.id };
  creations = () => page.nerve.to("/api/v0/projects/A/views").filter((call) => call.method === "POST");
  await page.open();
  expect(await saveAs()).toEqual({
    name: "Mine",
    description: "",
    logo_props: {},
    filters: CONDITIONS,
    display_filters: page.view.display_filters,
    display_properties: page.view.display_properties,
  });
});
