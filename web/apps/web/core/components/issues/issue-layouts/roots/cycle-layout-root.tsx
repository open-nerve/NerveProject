/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { CyclesDisabled } from "@/components/cycles/disabled";
import { CyclesReadFailure } from "@/components/cycles/read-failure";
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useCyclePreferencesFollow } from "@/hooks/use-cycle-preferences-follow";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { usePageFilterExpression } from "@/hooks/use-page-filter-expression";
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
 * page's own preview, whose host is the cycle's (re-66 L4).
 */
export const CycleLayoutRoot = observer(function CycleLayoutRoot() {
  // router
  const { workspaceSlug, projectId, cycleId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // hooks
  const follow = useCyclePreferencesFollow();
  const updateExpression = usePageFilterExpression(
    EIssuesStoreType.CYCLE,
    workspaceSlug,
    cycleId,
    issuesFilter,
    follow
  );
  // derived values
  const project = getProjectById(projectId);
  const cyclesOn = project?.cycle_view === true;
  const role =
    workspaceSlug && projectId ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  const preferences = useSessionSWR(
    role !== undefined && cyclesOn ? ["CYCLE_ISSUE_FILTERS", workspaceSlug, cycleId] : null,
    (slug, cycle) => issuesFilter.fetchFilters(slug, cycle)
  );
  const workItemFilters = cycleId ? issuesFilter.getIssueFilters(cycleId) : undefined;

  if (!workspaceSlug || !projectId || !cycleId || !project || role === undefined) return null;
  if (!cyclesOn) return <CyclesDisabled workspaceSlug={workspaceSlug} projectId={projectId} />;
  if (preferences.error !== undefined)
    return <CyclesReadFailure error={preferences.error} onRetry={() => void preferences.mutate()} />;
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
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
