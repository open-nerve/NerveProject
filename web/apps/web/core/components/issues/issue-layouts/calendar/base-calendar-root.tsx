import { followInSession } from "@/lib/in-session";
import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import { useCallback } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useCalendarView } from "@/hooks/store/use-calendar-view";
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
// types
import type { IQuickActionProps } from "../list/list-view-types";
import { CalendarChart } from "./calendar";
import { handleDragDrop } from "./utils";

export type CalendarStoreType = EIssuesStoreType.PROJECT;

interface IBaseCalendarRoot {
  QuickActions: FC<IQuickActionProps>;
  viewId?: string | undefined;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
}

export const BaseCalendarRoot = observer(function BaseCalendarRoot(props: IBaseCalendarRoot) {
  const { QuickActions, viewId, canEditPropertiesBasedOnProject } = props;

  // router
  const { workspaceSlug } = useParams();

  // hooks
  const storeType = useIssueStoreType() as CalendarStoreType;
  const { allowPermissions } = useUserPermissions();
  const { issues, issuesFilter, issueMap } = useIssues(storeType);
  const {
    fetchIssues,
    fetchNextIssues,
    createIssue,
    updateIssue,
    dropIssue,
    removeIssue,
    archiveIssue,
    restoreIssue,
    updateFilters,
  } = useIssuesActions(storeType);

  const issueCalendarView = useCalendarView();

  const isEditingAllowed = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT
  );

  const { enableInlineEditing } = issues?.viewFlags || {};

  const displayFilters = issuesFilter.issueFilters?.displayFilters;

  const groups = (issues.groups ?? new Map()) as Map<string, Group>;

  const layout = displayFilters?.calendar?.layout ?? "month";
  const { startDate, endDate } = issueCalendarView.getStartAndEndDate(layout) ?? {};
  const staleRefresh = useIssueListRefresh(
    issues.currentList,
    [...groups.keys()]
      .filter(
        (date) =>
          startDate &&
          endDate &&
          date >= startDate &&
          date <= endDate &&
          (displayFilters?.calendar?.show_weekends !== false ||
            ![0, 6].includes(new Date(`${date}T12:00:00Z`).getUTCDay()))
      )
      .map((group) => ({ group }))
  );

  useIssueListMount(
    storeType,
    issues.rootIssueStore,
    fetchIssues,
    startDate && endDate
      ? {
          limit: layout === "month" ? 4 : 30,
          query: { group_by: "target_date", target_date: [`${startDate}..${endDate}`] },
        }
      : undefined,
    viewId,
    layout
  );

  const handleDragAndDrop = async (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => {
    if (!issueId || !destinationDate || !sourceDate || !issueProjectId) return;

    await followInSession(
      () => handleDragDrop(issueId, sourceDate, destinationDate, workspaceSlug, issueProjectId, dropIssue),
      {
        failed: () => {
          setToast({
            title: "Error!",
            type: TOAST_TYPE.ERROR,
            message: "Failed to perform this action",
          });
        },
      }
    );
  };

  const loadMoreIssues = useCallback(
    (dateString: string) => {
      void fetchNextIssues(dateString).catch(() => {});
    },
    [fetchNextIssues]
  );

  const getPaginationData = useCallback(
    (groupId: string | undefined) => issues?.getPaginationData(groupId, undefined),
    [issues?.getPaginationData]
  );

  const getGroupIssueCount = useCallback(
    (groupId: string | undefined) => issues?.getGroupIssueCount(groupId, undefined, false),
    [issues?.getGroupIssueCount]
  );

  const canEditProperties = useCallback(
    (projectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && projectId ? canEditPropertiesBasedOnProject(projectId) : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  return (
    <>
      <IssueListRetry {...staleRefresh} />
      <div className="h-full w-full overflow-hidden bg-surface-1 pt-4">
        <CalendarChart
          issuesFilterStore={issuesFilter}
          issues={issueMap}
          groups={groups}
          layout={displayFilters?.calendar?.layout}
          showWeekends={displayFilters?.calendar?.show_weekends ?? false}
          issueCalendarView={issueCalendarView}
          quickActions={({ issue, parentRef, customActionButton, placement }) => (
            <QuickActions
              parentRef={parentRef}
              customActionButton={customActionButton}
              issue={issue}
              handleDelete={async () => removeIssue(issue.project_id, issue.id)}
              handleUpdate={async (data) => updateIssue && updateIssue(issue.project_id, issue.id, data)}
              handleArchive={async () => archiveIssue && archiveIssue(issue.project_id, issue.id)}
              handleRestore={async () => restoreIssue && restoreIssue(issue.project_id, issue.id)}
              readOnly={!canEditProperties(issue.project_id ?? undefined)}
              placements={placement}
            />
          )}
          loadMoreIssues={loadMoreIssues}
          getPaginationData={getPaginationData}
          getGroupIssueCount={getGroupIssueCount}
          quickAddCallback={createIssue}
          updateFilters={updateFilters}
          handleDragAndDrop={handleDragAndDrop}
          canEditProperties={canEditProperties}
        />
      </div>
    </>
  );
});
