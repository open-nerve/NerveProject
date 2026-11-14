import { followInSession, sessionGuard } from "@/lib/in-session";
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
import { EIssueFilterType, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useDropHighlight } from "@/hooks/use-drop-highlight";

import type { EIssuesStoreType } from "@nerve/types";
// hooks
import { useCalendarView } from "@/hooks/store/use-calendar-view";
import { useIssues } from "@/hooks/store/use-issues";
import { usePageIssueFilters } from "@/hooks/store/use-page-issue-filters";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
// types
import type { IQuickActionProps } from "../list/list-view-types";
import { CalendarChart } from "./calendar";
import type { TUpdateCalendarDisplay } from "./dropdowns";
import { handleDragDrop } from "./utils";

export type CalendarStoreType = EIssuesStoreType.PROJECT | EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE;

interface IBaseCalendarRoot {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  /**
   * The whole list is read-only, whatever the caller may do in the project (an ended or archived cycle's page): no
   * creation, inline edit or drag, and the rows' quick actions are read-only.
   */
  readOnly?: boolean;
}

export const BaseCalendarRoot = observer(function BaseCalendarRoot(props: IBaseCalendarRoot) {
  const { QuickActions, canEditPropertiesBasedOnProject, readOnly = false } = props;

  // router
  const { workspaceSlug, projectId } = useParams();

  // hooks
  const storeType = useIssueStoreType() as CalendarStoreType;
  const { allowPermissions } = useUserPermissions();
  const { issues, issueMap } = useIssues(storeType);
  const {
    fetchIssues,
    fetchNextIssues,
    createIssue,
    dropIssue,
    removeIssue,
    removeIssueFromView,
    archiveIssue,
    restoreIssue,
    updateFilters,
  } = useIssuesActions(storeType);

  const issueCalendarView = useCalendarView();

  const isEditingAllowed =
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );
  // A read-only list or a caller who may not write here: no quick add, and no drag or drop of a day's row.
  const isCalendarReadOnly = readOnly || !isEditingAllowed;

  const { enableInlineEditing } = issues?.viewFlags || {};

  const calendar = usePageIssueFilters(storeType)?.displayFilters?.calendar;

  const groups = (issues.groups ?? new Map()) as Map<string, Group>;

  const layout = calendar?.layout ?? "month";
  const showWeekends = calendar?.show_weekends ?? false;
  const { startDate, endDate } = issueCalendarView.getStartAndEndDate(layout) ?? {};
  const host = JSON.stringify([workspaceSlug, projectId, layout, startDate, endDate]);
  const capture = useDetailScope(workspaceSlug ?? "", projectId, host, true);
  const highlight = useDropHighlight(host);
  const refuse = useRefusalToast();
  const staleRefresh = useIssueListRefresh(
    issues.currentList,
    [...groups.keys()]
      .filter(
        (date) =>
          startDate &&
          endDate &&
          date >= startDate &&
          date <= endDate &&
          (calendar?.show_weekends !== false || ![0, 6].includes(new Date(`${date}T12:00:00Z`).getUTCDay()))
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
    undefined,
    layout
  );

  const handleDragAndDrop = async (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => {
    if (!issueId || !destinationDate || !sourceDate || !issueProjectId || !dropIssue || isCalendarReadOnly) return;
    if (sourceDate === destinationDate || issueProjectId !== projectId || issueMap[issueId]?.target_date !== sourceDate)
      return;
    const binding = issues.currentList;
    const sameHost = capture();
    const sameSession = sessionGuard();
    const current = () => sameHost() && sameSession() && issues.currentList === binding;
    if (!binding || !current()) return;
    await followInSession(
      () => handleDragDrop(issueId, sourceDate, destinationDate, workspaceSlug, issueProjectId, dropIssue),
      {
        done: () => {
          if (current()) highlight(`issue-${issueId}`, false, current);
        },
        failed: (error) => {
          if (current()) refuse(error);
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
    [issues]
  );

  const getGroupIssueCount = useCallback(
    (groupId: string | undefined) => issues?.getGroupIssueCount(groupId, undefined, false),
    [issues]
  );

  // A change of the page's calendar display keeps the rest of it; the page's own preferences take it.
  const updateCalendar: TUpdateCalendarDisplay | undefined = projectId
    ? (change) => {
        void updateFilters(projectId, EIssueFilterType.DISPLAY_FILTERS, { calendar: { ...calendar, ...change } });
      }
    : undefined;

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) => {
      if (readOnly) return false;
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && targetProjectId
          ? canEditPropertiesBasedOnProject(targetProjectId)
          : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed, readOnly]
  );

  return (
    <>
      <IssueListRetry {...staleRefresh} />
      <div className="h-full w-full overflow-hidden bg-surface-1 pt-4">
        <CalendarChart
          readOnly={isCalendarReadOnly}
          issues={issueMap}
          groups={groups}
          layout={layout}
          showWeekends={showWeekends}
          updateCalendar={updateCalendar}
          issueCalendarView={issueCalendarView}
          quickActions={({ issue, parentRef, customActionButton, placement }) => (
            <QuickActions
              parentRef={parentRef}
              customActionButton={customActionButton}
              issue={issue}
              handleDelete={async () => removeIssue(issue.project_id, issue.id)}
              handleRemoveFromView={
                removeIssueFromView && (async () => removeIssueFromView(issue.project_id, issue.id))
              }
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
          handleDragAndDrop={handleDragAndDrop}
          canEditProperties={canEditProperties}
        />
      </div>
    </>
  );
});
