/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EIssueLayoutTypes } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { PlanningDisabled } from "@/components/planning/disabled";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { PLANNING_PAGE_DOMAINS, type PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useIssues } from "@/hooks/store/use-issues";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { usePageFilterExpression } from "@/hooks/use-page-filter-expression";
import { usePlanningPagePreferencesFollow } from "@/hooks/use-planning-page-preferences-follow";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { PlanningPageCalendarLayout } from "../calendar/roots/planning-page-root";
import { PlanningPageKanBanLayout } from "../kanban/roots/planning-page-root";
import { PlanningPageListLayout } from "../list/roots/planning-page-root";
import { PLANNING_PAGE_LAYOUTS } from "../planning-page-layouts";
import { PreferencesRereadFailure } from "../preferences-reread-failure";
import { PlanningPageSpreadsheetLayout } from "../spreadsheet/roots/planning-page-root";

function PlanningPageIssueLayout(props: {
  storeType: PlanningPageStoreType;
  activeLayout: EIssueLayoutTypes | undefined;
}) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <PlanningPageListLayout storeType={props.storeType} />;
    case EIssueLayoutTypes.KANBAN:
      return <PlanningPageKanBanLayout storeType={props.storeType} />;
    case EIssueLayoutTypes.CALENDAR:
      return <PlanningPageCalendarLayout storeType={props.storeType} />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <PlanningPageSpreadsheetLayout storeType={props.storeType} />;
    default:
      return null;
  }
}

type Props = {
  /** The page's store type, which picks its domain's rows (PLANNING_PAGE_DOMAINS, PLANNING_PAGE_LAYOUTS). */
  storeType: PlanningPageStoreType;
  /**
   * What the page puts beside its own preview host, there once the host is (the cycle's page: the preview that the way
   * in from the card of the cycle in progress asked for, PeekOnEntry, P5b spec 2.7; re-70 F4).
   */
  children?: ReactNode;
};

/**
 * A planning page's work items (P5b spec 2.1, 2.6; P6b spec 2.2, 2.7), of the route's project and of the page's cycle
 * or module (the route param of its domain names it): the project's list with it, under the page's own preferences, by
 * the page's store type. It reads them for a member of the project while the domain is on; with it off it is the
 * domain's turned-off state and reads nothing, nor does its list (re-66 H1). It says why nerve could not give the
 * preferences first, with a retry; once given, a read of them again that fails leaves the list under the preferences
 * read, with a line above it that says so, with a retry (P6b spec 2.8, R4). Then the page's filters (their conditions
 * saved as its preferences, followed as the project page's), the row of their conditions, the layout they choose, and
 * the page's own preview, whose host is the page's (re-66 L4), with what the page puts beside it. Each difference of a
 * domain is a row of the tables, never a branch here.
 */
export const PlanningPageLayoutRoot = observer(function PlanningPageLayoutRoot({ storeType, children }: Props) {
  // router
  const params = useParams();
  const { workspaceSlug, projectId } = params;
  const { param, name } = PLANNING_PAGE_DOMAINS[storeType];
  const { domain, preferences: preferencesKey, filters } = PLANNING_PAGE_LAYOUTS[storeType];
  const entityId = params[param];
  // store hooks
  const { issues, issuesFilter } = useIssues(storeType);
  // hooks
  const follow = usePlanningPagePreferencesFollow(storeType);
  const updateExpression = usePageFilterExpression(storeType, workspaceSlug, entityId, issuesFilter, follow);
  // derived values: a member's read, of a project not archived (none under the project's wrapper is), while the
  // domain is on; with it off, the domain's turned-off state
  const { readable, listed } = usePlanningReads(workspaceSlug, projectId, domain);
  const preferences = useSessionSWR(listed ? [preferencesKey, workspaceSlug, entityId] : null, (slug, id) =>
    issuesFilter.fetchFilters(slug, id)
  );
  const workItemFilters = entityId ? issuesFilter.getIssueFilters(entityId) : undefined;
  const retry = () => void preferences.mutate();

  if (!workspaceSlug || !projectId || !entityId || !readable) return null;
  if (!listed) return <PlanningDisabled workspaceSlug={workspaceSlug} projectId={projectId} domain={domain} />;
  // nerve has not given the page's preferences yet: why its first read failed, or nothing while it is out
  if (!workItemFilters)
    return preferences.error === undefined ? null : <PlanningReadFailure error={preferences.error} onRetry={retry} />;

  return (
    <IssuesStoreContext.Provider value={storeType}>
      <ProjectLevelWorkItemFiltersHOC
        enableSaveView
        entityType={storeType}
        entityId={entityId}
        filtersToShowByLayout={filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={updateExpression}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: pageWorkItemsFilter }) => (
          <div className="relative flex h-full w-full flex-col overflow-hidden">
            {preferences.error !== undefined && <PreferencesRereadFailure error={preferences.error} onRetry={retry} />}
            {pageWorkItemsFilter && <WorkItemFiltersRow filter={pageWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto bg-surface-1">
              {/* mutation loader */}
              {issues.getIssueLoader() === "mutation" && (
                <div className="shadow-sm fixed top-[70px] right-[20px] z-20 flex h-[40px] w-[40px] items-center justify-center rounded-sm bg-layer-1">
                  <Spinner className="h-4 w-4" />
                </div>
              )}
              <PlanningPageIssueLayout storeType={storeType} activeLayout={workItemFilters.displayFilters?.layout} />
            </div>
            <IssuePeekOverview hostScope={JSON.stringify([name, workspaceSlug, projectId, entityId])} />
            {children}
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
