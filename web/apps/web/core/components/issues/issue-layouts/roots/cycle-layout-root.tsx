/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { PeekOnEntry } from "@/components/cycles/peek-on-entry";
import { PlanningDisabled } from "@/components/planning/disabled";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { usePageFilterExpression } from "@/hooks/use-page-filter-expression";
import { usePlanningPagePreferencesFollow } from "@/hooks/use-planning-page-preferences-follow";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { CycleCalendarLayout } from "../calendar/roots/cycle-root";
import { CycleKanBanLayout } from "../kanban/roots/cycle-root";
import { CycleListLayout } from "../list/roots/cycle-root";
import { CycleSpreadsheetLayout } from "../spreadsheet/roots/cycle-root";

function CycleIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined }) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <CycleListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <CycleKanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <CycleCalendarLayout />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <CycleSpreadsheetLayout />;
    default:
      return null;
  }
}

/**
 * A cycle's page's work items (P5b spec 2.1, 2.6), of the route's project and cycle: its project's list with the
 * cycle, under the page's own preferences, by the cycle's store type. It reads them for a member of the project while
 * its cycles are on; with them off it is the cycles' turned-off state and reads nothing, nor does its list (re-66 H1).
 * It says why nerve could not read the preferences, with a retry; then the page's filters (their conditions saved as
 * its preferences, followed as the project page's), the row of their conditions, the layout they choose, and the
 * page's own preview, whose host is the cycle's (re-66 L4), with the work item the way in asked it to preview
 * (PeekOnEntry: the card of the cycle in progress, P5b spec 2.7).
 */
export const CycleLayoutRoot = observer(function CycleLayoutRoot() {
  // router
  const { workspaceSlug, projectId, cycleId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  // hooks
  const follow = usePlanningPagePreferencesFollow(EIssuesStoreType.CYCLE);
  const updateExpression = usePageFilterExpression(
    EIssuesStoreType.CYCLE,
    workspaceSlug,
    cycleId,
    issuesFilter,
    follow
  );
  // derived values: a member's read, of a project not archived (none under the project's wrapper is), while its cycles
  // are on; with them off, the cycles' turned-off state
  const { readable, listed } = usePlanningReads(workspaceSlug, projectId, "cycles");
  const preferences = useSessionSWR(listed ? ["CYCLE_ISSUE_FILTERS", workspaceSlug, cycleId] : null, (slug, cycle) =>
    issuesFilter.fetchFilters(slug, cycle)
  );
  const workItemFilters = cycleId ? issuesFilter.getIssueFilters(cycleId) : undefined;

  if (!workspaceSlug || !projectId || !cycleId || !readable) return null;
  if (!listed) return <PlanningDisabled workspaceSlug={workspaceSlug} projectId={projectId} domain="cycles" />;
  if (preferences.error !== undefined)
    return <PlanningReadFailure error={preferences.error} onRetry={() => void preferences.mutate()} />;
  if (!workItemFilters) return null;

  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.CYCLE}>
      <ProjectLevelWorkItemFiltersHOC
        enableSaveView
        entityType={EIssuesStoreType.CYCLE}
        entityId={cycleId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.cycle_issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={updateExpression}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: cycleWorkItemsFilter }) => (
          <div className="relative flex h-full w-full flex-col overflow-hidden">
            {cycleWorkItemsFilter && <WorkItemFiltersRow filter={cycleWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto bg-surface-1">
              {/* mutation loader */}
              {issues.getIssueLoader() === "mutation" && (
                <div className="shadow-sm fixed top-[70px] right-[20px] z-20 flex h-[40px] w-[40px] items-center justify-center rounded-sm bg-layer-1">
                  <Spinner className="h-4 w-4" />
                </div>
              )}
              <CycleIssueLayout activeLayout={workItemFilters.displayFilters?.layout} />
            </div>
            <IssuePeekOverview hostScope={JSON.stringify(["cycle", workspaceSlug, projectId, cycleId])} />
            <PeekOnEntry workspaceSlug={workspaceSlug} projectId={projectId} />
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
