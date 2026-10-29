import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import { useCallback, useEffect } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve constants
import { EIssueFilterType, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
// types
import type { EIssuesStoreType, GroupByColumnTypes, TIssueKanbanFilters } from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
// components
import { IssueLayoutHOC } from "../issue-layout-HOC";
import { List } from "./default";
// types
import type { IQuickActionProps, TRenderQuickActions } from "./list-view-types";

type ListStoreType = EIssuesStoreType.PROJECT | EIssuesStoreType.PROFILE | EIssuesStoreType.ARCHIVED;

interface IBaseListRoot {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  viewId?: string | undefined;
}
export const BaseListRoot = observer(function BaseListRoot(props: IBaseListRoot) {
  const { QuickActions, viewId, canEditPropertiesBasedOnProject } = props;
  // router
  const storeType = useIssueStoreType() as ListStoreType;
  //stores
  const { issuesFilter, issues } = useIssues(storeType);
  const { fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, archiveIssue, restoreIssue } =
    useIssuesActions(storeType);
  // mobx store
  const { allowPermissions } = useUserPermissions();
  const { issueMap } = useIssues();

  const displayFilters = issuesFilter?.issueFilters?.displayFilters;
  const displayProperties = issuesFilter?.issueFilters?.displayProperties;
  const orderBy = displayFilters?.order_by || undefined;

  const group_by = (displayFilters?.group_by || null) as GroupByColumnTypes | null;
  const showEmptyGroup = displayFilters?.show_empty_groups ?? false;

  const { workspaceSlug, projectId } = useParams();
  const { updateFilters } = useIssuesActions(storeType);
  const collapsedGroups =
    issuesFilter?.issueFilters?.kanbanFilters || ({ group_by: [], sub_group_by: [] } as TIssueKanbanFilters);

  // The mounted binding hook shows failures and offers retry.
  useEffect(() => {
    void fetchIssues("init-loader", { limit: group_by ? 50 : 100 }, viewId).catch(() => {});
  }, [fetchIssues, storeType, group_by, viewId]);

  const groups = issues?.groups as Map<string, Group> | undefined;
  const staleRefresh = useIssueListRefresh(
    "currentList" in issues ? issues.currentList : undefined,
    [...(groups ?? [])]
      .filter(([id, group]) => !collapsedGroups.group_by.includes(id) && (showEmptyGroup || group.total > 0))
      .map(([group]) => ({ group }))
  );

  // auth
  const isEditingAllowed = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT
  );
  const { enableInlineEditing, enableQuickAdd, enableIssueCreation } = issues?.viewFlags || {};

  const canEditProperties = useCallback(
    (projectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && projectId ? canEditPropertiesBasedOnProject(projectId) : isEditingAllowed;

      return !!enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  const handleOnDrop = useGroupIssuesDragNDrop(storeType, orderBy, group_by);

  const renderQuickActions: TRenderQuickActions = useCallback(
    ({ issue, parentRef }) => (
      <QuickActions
        parentRef={parentRef}
        issue={issue}
        handleDelete={async () => removeIssue(issue.project_id, issue.id)}
        handleUpdate={async (data) => updateIssue && updateIssue(issue.project_id, issue.id, data)}
        handleArchive={async () => archiveIssue && archiveIssue(issue.project_id, issue.id)}
        handleRestore={async () => restoreIssue && restoreIssue(issue.project_id, issue.id)}
        readOnly={!canEditProperties(issue.project_id ?? undefined)}
      />
    ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canEditProperties, removeIssue, updateIssue, archiveIssue, restoreIssue]
  );

  const loadMoreIssues = useCallback(
    (groupId?: string) => {
      void fetchNextIssues(groupId).catch(() => {});
    },
    [fetchNextIssues]
  );

  // kanbanFilters and EIssueFilterType.KANBAN_FILTERS are used because the state is shared between kanban view and list view
  const handleCollapsedGroups = useCallback(
    (value: string) => {
      if (workspaceSlug) {
        let collapsedGroups = issuesFilter?.issueFilters?.kanbanFilters?.group_by || [];
        if (collapsedGroups.includes(value)) {
          collapsedGroups = collapsedGroups.filter((_value) => _value != value);
        } else {
          collapsedGroups.push(value);
        }
        updateFilters(projectId ?? "", EIssueFilterType.KANBAN_FILTERS, {
          group_by: collapsedGroups,
        } as TIssueKanbanFilters);
      }
    },
    [workspaceSlug, issuesFilter, projectId, updateFilters]
  );

  return (
    <IssueLayoutHOC layout={EIssueLayoutTypes.LIST}>
      <div className="relative size-full bg-surface-2">
        <IssueListRetry {...staleRefresh} />
        <List
          issuesMap={issueMap}
          displayProperties={displayProperties}
          group_by={group_by}
          orderBy={orderBy}
          updateIssue={updateIssue}
          quickActions={renderQuickActions}
          groups={groups ?? new Map()}
          loadMoreIssues={loadMoreIssues}
          showEmptyGroup={showEmptyGroup}
          quickAddCallback={createIssue}
          enableIssueQuickAdd={!!enableQuickAdd}
          canEditProperties={canEditProperties}
          disableIssueCreation={!enableIssueCreation || !isEditingAllowed}
          handleOnDrop={handleOnDrop}
          handleCollapsedGroups={handleCollapsedGroups}
          collapsedGroups={collapsedGroups}
        />
      </div>
    </IssueLayoutHOC>
  );
});
