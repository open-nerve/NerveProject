/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { observer } from "mobx-react";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import type { TWorkItemFilterExpression } from "@nerve/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { useProjectViewPreferencesFollow } from "@/hooks/use-project-view-preferences-follow";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { CalendarLayout } from "../calendar/roots/project-root";
import { KanBanLayout } from "../kanban/roots/project-root";
import { ListLayout } from "../list/roots/project-root";
import { ProjectSpreadsheetLayout } from "../spreadsheet/roots/project-root";

function ProjectViewIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined }) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <ListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <KanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <CalendarLayout />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <ProjectSpreadsheetLayout />;
    default:
      return null;
  }
}

type Props = {
  /** The page's address, as the route gives it: the project, and its view. */
  workspaceSlug: string;
  projectId: string;
  viewId: string;
};

/**
 * A project's view's page's work items (P6a spec 2.5, 2.6, R1, R10), of the route's project and view: the project's
 * list under the page's draft of the view, by the view page's store type, once the page's read has seeded the draft.
 * Then the page's filters, the row of their conditions, the layout the draft chooses, with the project page's layouts
 * and rows, and the page's own preview, whose host is the page's. A change of the filters is the draft's alone,
 * followed as the project page's (the list read again under it): nothing is sent, and the filters' saved baseline
 * stays the view's, so the page holds a change of the view (usePageFilterExpression, which advances it to what nerve
 * confirmed, is the pages' whose preferences are saved, not this one's), which Update view saves, or Save view as a
 * new view (P6a spec 2.7, R9).
 */
export const ProjectViewLayoutRoot = observer(function ProjectViewLayoutRoot(props: Props) {
  const { workspaceSlug, projectId, viewId } = props;
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  // hooks
  const follow = useProjectViewPreferencesFollow();
  const updateFilters = useCallback(
    (expression: TWorkItemFilterExpression) =>
      void follow(workspaceSlug, viewId, () => issuesFilter.updateFilterExpression(workspaceSlug, viewId, expression)),
    [follow, issuesFilter, workspaceSlug, viewId]
  );
  // derived values
  const workItemFilters = issuesFilter.getIssueFilters(viewId);

  if (!workItemFilters) return null;

  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.PROJECT_VIEW}>
      <ProjectLevelWorkItemFiltersHOC
        entityType={EIssuesStoreType.PROJECT_VIEW}
        entityId={viewId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={updateFilters}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
        enableSaveView
        enableUpdateView
        preferences={issuesFilter}
      >
        {({ filter: viewWorkItemsFilter }) => (
          <div className="relative flex h-full w-full flex-col overflow-hidden">
            {viewWorkItemsFilter && <WorkItemFiltersRow filter={viewWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto bg-surface-1">
              {/* mutation loader */}
              {issues.getIssueLoader() === "mutation" && (
                <div className="shadow-sm fixed top-[70px] right-[20px] z-20 flex h-[40px] w-[40px] items-center justify-center rounded-sm bg-layer-1">
                  <Spinner className="h-4 w-4" />
                </div>
              )}
              <ProjectViewIssueLayout activeLayout={workItemFilters.displayFilters?.layout} />
            </div>
            <IssuePeekOverview hostScope={JSON.stringify(["project-view", workspaceSlug, projectId, viewId])} />
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
