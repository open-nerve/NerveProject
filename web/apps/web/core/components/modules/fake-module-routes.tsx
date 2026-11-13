// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// The pages a row of a module's progress is on (P6b spec 2.7, R14), for their tests: project A's modules list with the
// preview of a module (?peekModule), and M1's own page, its work items and its progress sidebar, under one router.
// Project A has the caller as a member and its labels Bug and UI; M1's progress has a row of each kind: state groups,
// an assignee and none, the two labels and none (fake-progress-rows.ts, where the tests find the rows). The stores,
// routes and HTTP are real; the test file stands in for what the rows do not reach (the burndown chart, the sidebar's
// head, details and links).
import { act } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { SWRConfig } from "swr";
import type { ModuleIssueFilters } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import type { Call } from "@/lib/auth/fake-nerve";
import { loadRowLabels, progressRows } from "@/components/core/sidebar/fake-progress-rows";
import {
  M1,
  PlanningPageWorkItems,
  layouts,
  modulePreferencesOf,
  planningPageTab,
} from "@/components/issues/issue-layouts/roots/fake-planning-page";
import { loadModuleProgress, moduleProgressOf, moduleRequests } from "@/store/planning/fake-modules";
import { ModulePeekOverview } from "./module-peek-overview";
import { ModuleDetailsSidebar } from "./progress-sidebar";

export { M1 };
export const LIST = "/ws/projects/A/modules";
export const PAGE = `/ws/projects/A/modules/${M1}`;

/** The module's page, of the route's module: its work items and its progress sidebar. */
function ModulePage() {
  return (
    <>
      <PlanningPageWorkItems type={EIssuesStoreType.MODULE} />
      <aside aria-label="sidebar">
        <ModuleDetailsSidebar handleClose={() => undefined} workspaceSlug="ws" projectId="A" moduleId={M1} />
      </aside>
    </>
  );
}

/** The project's modules list: the preview of the module ?peekModule names. */
function ListPage() {
  return (
    <aside aria-label="preview">
      <ModulePeekOverview workspaceSlug="ws" projectId="A" />
    </aside>
  );
}

/**
 * A tab at project A's modules: its module M1, of the fields given (planningPageTab's), and its progress, its links
 * (none) and M1 itself as nerve answers them each time; its labels. Nothing else is answered: the test answers the
 * page's preferences and its lists.
 */
export async function moduleRoutesTab(fields: Parameters<typeof planningPageTab>[1] = {}) {
  const tab = await planningPageTab(EIssuesStoreType.MODULE, fields);
  const module = tab.app.planning.modules.getModuleById(M1)!;
  const progress = moduleProgressOf(progressRows);
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    if (call.path === moduleRequests.progress(module)) call.answer(json(200, progress));
    else if (call.path === moduleRequests.links(module)) call.answer(json(200, { data: [] }));
    else if (call.path === moduleRequests.at(module)) call.answer(json(200, module));
    else states?.(call);
  };
  await loadModuleProgress(tab.nerve, tab.app.planning.modules, module, progress);
  await loadRowLabels(tab);
  tab.nerve.calls.length = 0;
  const cache = new Map();
  return {
    ...tab,
    /** The changes of the page's preferences sent so far. */
    patches: () => tab.preferences().filter((call) => call.method === "PATCH"),
    /** The reads of the page's preferences sent so far. */
    reads: () => tab.preferences().filter((call) => call.method === "GET"),
    /** Renders the pages at the address at. */
    open: (at: string) =>
      tab.render(
        <SWRConfig value={{ provider: () => cache }}>
          <MemoryRouter initialEntries={[at]}>
            <Routes>
              <Route path=":workspaceSlug/projects/:projectId/modules" element={<ListPage />} />
              <Route path=":workspaceSlug/projects/:projectId/modules/:moduleId" element={<ModulePage />} />
            </Routes>
          </MemoryRouter>
        </SWRConfig>
      ),
  };
}

/** nerve answers call with the module page's preferences: a list grouped by state, with filters. */
export const answerPreferences = (call: Call, filters: ModuleIssueFilters = {}) =>
  act(async () => call.answer(json(200, modulePreferencesOf(layouts.list.display, filters))));
