// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// "Save view" collapsed (P4 spec 2.8, R17): the filters row of the project's work items page, a cycle's and a module's
// offers no "Save view", even with conditions in it, the project's views on and an admin to save it: nerve has no
// views until P6a, which brings it back with them. The row's other operations stay. The pages' work items, the stores
// and HTTP are real; the preview is left out.
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

let close: (() => Promise<void>) | undefined;
afterEach(async () => {
  await close?.();
  close = undefined;
});

/** A cycle's or a module's page of A, shown with the page's conditions. */
async function planningPage(type: EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE) {
  const page = await planningPageTab(type, ADMIN);
  bound.app = page.app;
  close = page.close;
  const cycle = type === EIssuesStoreType.CYCLE;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...(cycle ? { cycleId: C1 } : { moduleId: M1 }) };
  const preferences = (cycle ? cyclePreferencesOf : modulePreferencesOf)(layouts.list.display, CONDITIONS);
  await page.show(preferences, page.listOf("list"));
}

/** The project A's work items page, shown with the page's conditions. */
async function projectPage() {
  const tab = await issueSelectionTab();
  bound.app = tab.app;
  close = tab.close;
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({
    ...project,
    member_role: ADMIN.role,
    ...ADMIN.project,
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
  ["the project's page", projectPage],
  ["a cycle's page", () => planningPage(EIssuesStoreType.CYCLE)],
  ["a module's page", () => planningPage(EIssuesStoreType.MODULE)],
])("offers no Save view on %s, its conditions shown to an admin, its project's views on", async (_, open) => {
  await open();
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(buttonOf("Clear all"), "the row of the page's conditions").toBeDefined();
  expect(buttonOf("Save view")).toBeUndefined();
});
