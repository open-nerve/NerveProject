/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useUserPermissions } from "@/hooks/store/user";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
// nerve constants
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import { useProjectPreferencesFollow } from "@/hooks/use-project-preferences-follow";
import type { TWorkItemFilterExpression } from "@nerve/types";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { CalendarLayout } from "../calendar/roots/project-root";
import { KanBanLayout } from "../kanban/roots/project-root";
import { ListLayout } from "../list/roots/project-root";
import { ProjectSpreadsheetLayout } from "../spreadsheet/roots/project-root";

function ProjectIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined }) {
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

export const ProjectLayoutRoot = observer(function ProjectLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams();
  // hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROJECT);
  const follow = useProjectPreferencesFollow();
  const { getFilter } = useWorkItemFilters();
  const applyingConfirmation = useRef(false);
  const editSequence = useRef(0);
  const updateExpression = useCallback(
    (expression: TWorkItemFilterExpression) => {
      if (applyingConfirmation.current || !workspaceSlug || !projectId) return;
      const sequence = ++editSequence.current;
      const filter = getFilter(EIssuesStoreType.PROJECT, projectId);
      const submitted = JSON.stringify(filter?.expression);
      void follow(workspaceSlug, projectId, async () => {
        await issuesFilter.updateFilterExpression(workspaceSlug, projectId, expression);
        // Capture the entire draft, including unfinished inputs that have not notified a save yet.
        if (
          sequence === editSequence.current &&
          filter &&
          getFilter(EIssuesStoreType.PROJECT, projectId) === filter &&
          JSON.stringify(filter.expression) === submitted
        ) {
          applyingConfirmation.current = true;
          try {
            filter.resetExpression(issuesFilter.getIssueFilters(projectId)?.richFilters ?? {});
          } finally {
            applyingConfirmation.current = false;
          }
        }
      });
    },
    [workspaceSlug, projectId, follow, issuesFilter, getFilter]
  );
  // derived values
  const workItemFilters = projectId ? issuesFilter?.getIssueFilters(projectId) : undefined;
  const activeLayout = workItemFilters?.displayFilters?.layout;

  const permission = useUserPermissions();
  const role =
    workspaceSlug && projectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;
  const allowed = role !== undefined;
  const { t } = useTranslation();
  const { error } = useSessionSWR(
    allowed ? ["PROJECT_ISSUE_FILTERS", workspaceSlug, projectId] : null,
    (slug, project) => issuesFilter.fetchFilters(slug, project)
  );
  if (error) return <div role="alert">{t(errorMessageKey(error))}</div>;
  if (!allowed) return null;

  if (!workspaceSlug || !projectId || !workItemFilters) return <></>;
  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.PROJECT}>
      <ProjectLevelWorkItemFiltersHOC
        enableSaveView
        entityType={EIssuesStoreType.PROJECT}
        entityId={projectId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={updateExpression}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: projectWorkItemsFilter }) => (
          <div className="relative flex h-full w-full flex-col overflow-hidden">
            {projectWorkItemsFilter && <WorkItemFiltersRow filter={projectWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto bg-surface-1">
              {/* mutation loader */}
              {issues?.getIssueLoader() === "mutation" && (
                <div className="shadow-sm fixed top-[70px] right-[20px] z-20 flex h-[40px] w-[40px] items-center justify-center rounded-sm bg-layer-1">
                  <Spinner className="h-4 w-4" />
                </div>
              )}
              <ProjectIssueLayout activeLayout={activeLayout} />
            </div>
            {/* peek overview */}
            <IssuePeekOverview hostScope={JSON.stringify(["project", workspaceSlug, projectId])} />
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
