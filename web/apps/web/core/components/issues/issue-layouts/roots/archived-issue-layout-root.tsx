/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useUserPermissions } from "@/hooks/store/user";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
// components
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
// hooks
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
import { useIssues } from "@/hooks/store/use-issues";
import { useLocalIssueFilterUpdate } from "@/hooks/use-local-issue-filter-update";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { ArchivedIssueListLayout } from "../list/roots/archived-issue-root";

export const ArchivedIssueLayoutRoot = observer(function ArchivedIssueLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams();
  // hooks
  const { issuesFilter } = useIssues(EIssuesStoreType.ARCHIVED);
  const update = useLocalIssueFilterUpdate(workspaceSlug ?? "", projectId, projectId ?? "");
  // derived values
  const workItemFilters = projectId ? issuesFilter?.getIssueFilters(projectId) : undefined;

  const permission = useUserPermissions();
  const role =
    workspaceSlug && projectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;
  const allowed = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  const { t } = useTranslation();
  const { error } = useSessionSWR(
    allowed ? ["ARCHIVED_ISSUE_FILTERS", workspaceSlug, projectId] : null,
    (slug, project) => issuesFilter.fetchFilters(slug, project)
  );
  if (error) return <div role="alert">{t(errorMessageKey(error))}</div>;
  if (!allowed) return null;

  if (!workspaceSlug || !projectId || !workItemFilters) return <></>;
  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.ARCHIVED}>
      <ProjectLevelWorkItemFiltersHOC
        entityType={EIssuesStoreType.ARCHIVED}
        entityId={projectId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.archived_issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={(expression) => {
          void update(() => issuesFilter.updateFilterExpression(workspaceSlug, projectId, expression));
        }}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: archivedWorkItemsFilter }) => (
          <>
            {archivedWorkItemsFilter && <WorkItemFiltersRow filter={archivedWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto">
              <ArchivedIssueListLayout />
            </div>
            <IssuePeekOverview hostScope={JSON.stringify(["archives", workspaceSlug, projectId])} />
          </>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
