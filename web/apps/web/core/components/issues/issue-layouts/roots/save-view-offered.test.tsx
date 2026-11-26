// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// "Save view" offered (P6a spec 2.2, 2.7, R7; P4 spec 2.8, R17 collapsed it until nerve had views): the filters row of
// the project's work items page, a cycle's and a module's offers "Save view" with conditions in it to whom nerve lets
// make a view (viewsCreatableBy): an admin or a member, the project's views on; never a guest, nor with the views off.
// The row's other operations stay. Its words are keys (fake-i18n). The pages' work items, the stores and HTTP are
// real; the preview is left out.
import { act } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  buttonOf,
  C1,
  cyclePreferencesOf,
  layouts,
  M1,
  modulePreferencesOf,
  planningPageTab,
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
// jsdom has no viewport geometry; the filters row and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("../../peek-overview", () => ({ IssuePeekOverview: () => null }));

/** The page's conditions: its work items of high priority. */
const CONDITIONS = { priority: ["high"] };
/** The project A as an admin holds it, its views on. */
const ADMIN = { role: 20, project: { issue_views_view: true } } as const;
/** The project A as a guest holds it, its views on. */
const GUEST = { role: 5, project: { issue_views_view: true } } as const;
/** The project A as an admin holds it, its views off. */
const VIEWS_OFF = { role: 20, project: { issue_views_view: false } } as const;
type Caller = typeof ADMIN | typeof GUEST | typeof VIEWS_OFF;

let close: (() => Promise<void>) | undefined;
afterEach(async () => {
  await close?.();
  close = undefined;
});

/** A cycle's or a module's page of A, shown with the page's conditions to caller (an admin unless given). */
async function planningPage(type: EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE, caller: Caller = ADMIN) {
  const page = await planningPageTab(type, caller);
  bound.app = page.app;
  close = page.close;
  const cycle = type === EIssuesStoreType.CYCLE;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...(cycle ? { cycleId: C1 } : { moduleId: M1 }) };
  const preferences = (cycle ? cyclePreferencesOf : modulePreferencesOf)(layouts.list.display, CONDITIONS);
  await page.show(preferences, page.listOf("list"));
}

/** The project A's work items page, shown with the page's conditions to caller (an admin unless given). */
async function projectPage(caller: Caller = ADMIN) {
  const tab = await issueSelectionTab();
  bound.app = tab.app;
  close = tab.close;
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({
    ...project,
    member_role: caller.role,
    ...caller.project,
  }));
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const read = preferencesOf({ filters: CONDITIONS });
  read.display_filters = { ...read.display_filters, ...layouts.list.display };
  const preferences = () => tab.nerve.to("/api/v0/me/projects/A/preferences");
  await until(() => preferences().length === 1, "the page's preferences");
  await act(async () => preferences()[0]!.answer(json(200, read)));
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  await until(() => lists().length === 1, "the page's list");
  await act(async () => lists()[0]!.answer(json(200, layouts.list.page())));
}

it.each([
  ["the project's page", () => projectPage()],
  ["a cycle's page", () => planningPage(EIssuesStoreType.CYCLE)],
  ["a module's page", () => planningPage(EIssuesStoreType.MODULE)],
])("offers Save view on %s, its conditions shown to an admin, its project's views on", async (_, open) => {
  await open();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(buttonOf("common.clear_all"), "the row of the page's conditions").toBeDefined();
  expect(buttonOf("common.filter_options.save_view")).toBeDefined();
});

it.each([
  ["to a guest, its project's views on", () => projectPage(GUEST)],
  ["with its project's views off, to an admin", () => projectPage(VIEWS_OFF)],
  [
    "of a cycle's page with its project's views off, to an admin",
    () => planningPage(EIssuesStoreType.CYCLE, VIEWS_OFF),
  ],
])("offers no Save view %s, the page's conditions shown", async (_, open) => {
  await open();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(buttonOf("common.clear_all"), "the row of the page's conditions").toBeDefined();
  expect(buttonOf("common.filter_options.save_view")).toBeUndefined();
});
