// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// The pages a row of a cycle's progress leads from and to (P5b spec 2.7, 2.8), for their tests: project A's cycles list,
// with the card of its cycle C1 in progress and the preview of a cycle (?peekCycle), and C1's own page, its work items
// and its progress sidebar, under one router whose address the test reads and moves (where); any other address is a
// page elsewhere. Project A has the caller as a member and its labels Bug and UI; C1's progress has a row of each kind:
// state groups, an assignee and none, the two labels and none (fake-progress-rows.ts, where the tests find the rows).
// The stores, routes and HTTP are real; the test file stands in for what the rows do not reach (the cycle's list row,
// the burndown chart, the sidebar's head and details).
import { act } from "react";
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from "react-router";
import type { Location, NavigateFunction } from "react-router";
import { SWRConfig } from "swr";
import type { CyclePreferences, ProjectIssueFilters } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { answered, json } from "@/lib/auth/fake-nerve";
import type { Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadRowLabels, progressRows } from "@/components/core/sidebar/fake-progress-rows";
import {
  C1,
  PlanningPageWorkItems,
  cyclePreferencesOf,
  layouts,
  planningPageTab,
} from "@/components/issues/issue-layouts/roots/fake-planning-page";
import { loadCycles, progressOf, rowOf } from "@/store/planning/fake-cycles";
import { ActiveCycleRoot } from "./active-cycle/root";
import { CyclePeekOverview } from "./cycle-peek-overview";
import { CycleDetailsSidebar } from "./progress-sidebar";

export { C1 };
export const LIST = "/ws/projects/A/cycles";
export const PAGE = `/ws/projects/A/cycles/${C1}`;

/** Where the pages are: the router's address, and its navigate, which moves it as a link or a Back would. */
export const where: { location?: Location; navigate?: NavigateFunction } = {};
function Where() {
  where.location = useLocation();
  where.navigate = useNavigate();
  return null;
}

/** The cycle's page, of the route's cycle: its work items and its progress sidebar. */
function CyclePage() {
  return (
    <>
      <PlanningPageWorkItems type={EIssuesStoreType.CYCLE} />
      <aside aria-label="sidebar">
        <CycleDetailsSidebar handleClose={() => undefined} workspaceSlug="ws" projectId="A" cycleId={C1} />
      </aside>
    </>
  );
}

/** The project's cycles list: the card of its cycle in progress, and the preview of the cycle ?peekCycle names. */
function ListPage() {
  return (
    <>
      <ActiveCycleRoot workspaceSlug="ws" projectId="A" />
      <aside aria-label="preview">
        <CyclePeekOverview workspaceSlug="ws" projectId="A" />
      </aside>
    </>
  );
}

/**
 * A tab at project A's cycles: its cycle C1 in progress, listed, and its progress as nerve answers it each time; its
 * labels. Nothing else is answered: the test answers the page's preferences and its lists.
 */
export async function cycleRoutesTab() {
  const tab = await planningPageTab(EIssuesStoreType.CYCLE);
  const cycle = tab.app.planning.cycles.getCycleById(C1)!;
  const progress = progressOf(cycle, progressRows);
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    if (call.path === `/api/v0/cycles/${C1}/progress`) call.answer(json(200, progress));
    else if (call.path === `/api/v0/cycles/${C1}`) call.answer(json(200, cycle));
    else states?.(call);
  };
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(cycle)], tab.project);
  await answered(
    tab.nerve,
    () => tab.app.planning.cycles.fetchProgress(C1),
    ["GET", `/api/v0/cycles/${C1}/progress`],
    progress,
    "the progress"
  );
  await loadRowLabels(tab);
  tab.nerve.calls.length = 0;
  const cache = new Map();
  const preferences = () => tab.nerve.to(`/api/v0/me/cycles/${C1}/preferences`);
  return {
    ...tab,
    preferences,
    /** The changes of the page's preferences sent so far. */
    patches: () => preferences().filter((call) => call.method === "PATCH"),
    /** The reads of the page's preferences sent so far. */
    reads: () => preferences().filter((call) => call.method === "GET"),
    /** The reads of the project's work items sent so far: the cycle page's list, the card's priority work items. */
    lists: () => tab.nerve.to("/api/v0/projects/A/issues"),
    /** Renders the pages at the address at. */
    open: (at: string) =>
      tab.render(
        <SWRConfig value={{ provider: () => cache }}>
          <MemoryRouter initialEntries={[at]}>
            <Where />
            <Routes>
              <Route path=":workspaceSlug/projects/:projectId/cycles" element={<ListPage />} />
              <Route path=":workspaceSlug/projects/:projectId/cycles/:cycleId" element={<CyclePage />} />
              <Route path="*" element={null} />
            </Routes>
          </MemoryRouter>
        </SWRConfig>
      ),
    /** Moves the address to, as a link or a Back would. */
    go: (to: string) => act(async () => void where.navigate?.(to)),
  };
}

/** The cycle page's preferences as nerve answers them: a list grouped by state, with filters. */
export const answerOf = (filters: ProjectIssueFilters = {}): CyclePreferences =>
  cyclePreferencesOf(layouts.list.display, filters);

/** nerve answers call with the page's preferences of filters. */
export const answerPreferences = (call: Call, filters: ProjectIssueFilters = {}) =>
  act(async () => call.answer(json(200, answerOf(filters))));

/** Waits until the router's address is path. */
export const arrived = (path: string) => until(() => where.location?.pathname === path, `the address ${path}`);
