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
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
// components
import { IssueLayoutHOC } from "../issue-layout-HOC";
import { List } from "./default";
import { RowExpansionProvider } from "../row-expansion";
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

  useIssueListMount(
    storeType,
    issues.rootIssueStore,
    fetchIssues,
    { limit: group_by ? 50 : 100 },
    viewId,
    String(group_by)
  );

  const groups = issues?.groups as Map<string, Group> | undefined;
  const staleRefresh = useIssueListRefresh(
    "currentList" in issues ? issues.currentList : undefined,
    [...(groups ?? [])]
      .filter(([id, group]) => !collapsedGroups.group_by.includes(id) && (showEmptyGroup || group.total > 0))
      .map(([group]) => ({ group }))
  );

  // auth
  const isEditingAllowed =
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );
  const { enableInlineEditing, enableQuickAdd, enableIssueCreation } = issues?.viewFlags || {};

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && targetProjectId
          ? canEditPropertiesBasedOnProject(targetProjectId)
          : isEditingAllowed;

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
    [QuickActions, canEditProperties, removeIssue, updateIssue, archiveIssue, restoreIssue]
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
        let nextCollapsedGroups = issuesFilter?.issueFilters?.kanbanFilters?.group_by || [];
        if (nextCollapsedGroups.includes(value)) {
          nextCollapsedGroups = nextCollapsedGroups.filter((_value) => _value != value);
        } else {
          nextCollapsedGroups.push(value);
        }
        updateFilters(projectId ?? "", EIssueFilterType.KANBAN_FILTERS, {
          group_by: nextCollapsedGroups,
        } as TIssueKanbanFilters);
      }
    },
    [workspaceSlug, issuesFilter, projectId, updateFilters]
  );

  return (
    <RowExpansionProvider viewId={viewId}>
      <IssueListRetry {...staleRefresh} />
      <IssueLayoutHOC layout={EIssueLayoutTypes.LIST}>
        <div className="relative size-full bg-surface-2">
          <List
            viewId={viewId}
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
    </RowExpansionProvider>
  );
});
