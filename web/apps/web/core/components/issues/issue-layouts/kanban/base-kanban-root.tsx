import type { TProfileViews } from "@nerve/types";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import type { Issue } from "@nerve/api-client";
import { useCallback, useEffect, useRef, useState } from "react";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { EIssueFilterType, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
//hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
import { SessionChangedError } from "@/lib/auth/token-manager";
// types
import { DeleteIssueModal } from "../../delete-issue-modal";
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { IQuickActionProps, TRenderQuickActions } from "../list/list-view-types";
//components
import { KanBan } from "./default";
import { KanbanDeleteDropZone } from "./kanban-delete-drop-zone";
import { KanBanSwimLanes } from "./swimlanes";

type KanbanStoreType = EIssuesStoreType.PROJECT | EIssuesStoreType.PROFILE;

export interface IBaseKanBanLayout {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  viewId?: TProfileViews;
}

// Stable identity so an empty kanban-filter value doesn't churn the per-column cache's sharedDeps.
const EMPTY_KANBAN_FILTERS = { group_by: [], sub_group_by: [] };

export const BaseKanBanRoot = observer(function BaseKanBanRoot(props: IBaseKanBanLayout) {
  const { QuickActions, canEditPropertiesBasedOnProject, viewId } = props;
  // router
  const { workspaceSlug, projectId, userId } = useParams();
  // store hooks
  const storeType = useIssueStoreType() as KanbanStoreType;
  const { allowPermissions } = useUserPermissions();
  const { issueMap, issuesFilter, issues } = useIssues(storeType);
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const {
    fetchIssues,
    fetchNextIssues,
    createIssue,
    updateIssue,
    removeIssue,
    archiveIssue,
    restoreIssue,
    updateFilters,
  } = useIssuesActions(storeType);

  const localFilters =
    storeType === EIssuesStoreType.PROFILE
      ? issues.rootIssueStore.profileIssuesFilter.getIssueFilters(userId ?? "", workspaceSlug)
      : issuesFilter.issueFilters;
  const displayFilters = localFilters?.displayFilters;
  const displayProperties = localFilters?.displayProperties;

  const sub_group_by = displayFilters?.sub_group_by;
  const group_by = displayFilters?.group_by;

  const orderBy = displayFilters?.order_by;

  useIssueListMount(
    storeType,
    issues.rootIssueStore,
    fetchIssues,
    { limit: sub_group_by ? 10 : 30 },
    viewId,
    JSON.stringify([group_by, sub_group_by])
  );

  const fetchMoreIssues = useCallback(
    (groupId?: string, subgroupId?: string) => {
      if (issues?.getIssueLoader(groupId, subgroupId) !== "pagination") {
        void fetchNextIssues(groupId, subgroupId).catch(() => {});
      }
    },
    [fetchNextIssues, issues]
  );

  const groups = issues?.groups;
  const collapsed = localFilters?.kanbanFilters ?? EMPTY_KANBAN_FILTERS;
  const staleRefresh = useIssueListRefresh(
    issues.currentList,
    [...(groups ?? [])].flatMap(([group, data]) => {
      if (collapsed.group_by.includes(group) || (!displayFilters?.show_empty_groups && !data.total)) return [];
      return data.subgroups
        ? [...data.subgroups.keys()]
            .filter((subgroup) => !collapsed.sub_group_by.includes(subgroup))
            .map((subgroup) => ({ group, subgroup }))
        : [{ group }];
    })
  );

  const userDisplayFilters = displayFilters || null;

  const KanBanView = sub_group_by ? KanBanSwimLanes : KanBan;

  const { enableInlineEditing, enableQuickAdd, enableIssueCreation } = issues?.viewFlags || {};

  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);
  const [scrollableContainer, setScrollableContainer] = useState<HTMLDivElement | null>(null);
  const attachScrollableContainer = useCallback((element: HTMLDivElement | null) => {
    scrollableContainerRef.current = element;
    setScrollableContainer(element);
  }, []);

  // states
  // The dialog owns this selected snapshot through confirmation, including canonical removal on success.
  const [issueToDelete, setIssueToDelete] = useState<Issue>();

  const isEditingAllowed =
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );

  const handleOnDrop = useGroupIssuesDragNDrop(storeType, orderBy, group_by, sub_group_by);

  const handleRequestDelete = useCallback(
    (workItemId: string | undefined) => {
      setIssueToDelete(workItemId ? getIssueById(workItemId) : undefined);
    },
    [getIssueById]
  );

  const canEditProperties = useCallback(
    (issueProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && issueProjectId
          ? canEditPropertiesBasedOnProject(issueProjectId)
          : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  // The loading/verification gate can mount or replace the actual container later.
  useEffect(() => {
    if (!scrollableContainer) return;
    return autoScrollForElements({ element: scrollableContainer });
  }, [scrollableContainer]);

  const renderQuickActions: TRenderQuickActions = useCallback(
    ({ issue, parentRef, customActionButton }) => (
      <QuickActions
        parentRef={parentRef}
        customActionButton={customActionButton}
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

  const handleDeleteIssue = async () => {
    if (!issueToDelete) throw new SessionChangedError();
    // The confirmation owns refusal, pending and current-host completion.
    await removeIssue(issueToDelete.project_id, issueToDelete.id);
  };

  const handleCollapsedGroups = useCallback(
    (toggle: "group_by" | "sub_group_by", value: string) => {
      if (workspaceSlug) {
        let collapsedGroups = localFilters?.kanbanFilters?.[toggle] || [];
        if (collapsedGroups.includes(value)) {
          collapsedGroups = collapsedGroups.filter((_value) => _value != value);
        } else {
          collapsedGroups.push(value);
        }
        updateFilters(projectId ?? "", EIssueFilterType.KANBAN_FILTERS, {
          [toggle]: collapsedGroups,
        });
      }
    },
    [workspaceSlug, localFilters?.kanbanFilters, projectId, updateFilters]
  );

  const collapsedGroups = localFilters?.kanbanFilters || EMPTY_KANBAN_FILTERS;

  return (
    <>
      <IssueListRetry {...staleRefresh} />
      <DeleteIssueModal
        data={issueToDelete}
        isOpen={!!issueToDelete}
        handleClose={() => setIssueToDelete(undefined)}
        onSubmit={handleDeleteIssue}
      />
      <KanbanDeleteDropZone onRequestDelete={handleRequestDelete} />
      <IssueLayoutHOC layout={EIssueLayoutTypes.KANBAN}>
        <div
          className={`horizontal-scrollbar relative scrollbar-lg flex h-full w-full bg-surface-2 ${sub_group_by ? "vertical-scrollbar overflow-y-auto" : "overflow-x-auto overflow-y-hidden"}`}
          ref={attachScrollableContainer}
        >
          <div className="relative h-full w-max min-w-full bg-surface-2">
            <div className="h-full w-max">
              <KanBanView
                issuesMap={issueMap}
                groups={groups ?? new Map()}
                getGroupIssueCount={issues.getGroupIssueCount}
                displayProperties={displayProperties}
                sub_group_by={sub_group_by}
                group_by={group_by}
                orderBy={orderBy}
                updateIssue={updateIssue}
                quickActions={renderQuickActions}
                handleCollapsedGroups={handleCollapsedGroups}
                collapsedGroups={collapsedGroups}
                enableQuickIssueCreate={enableQuickAdd}
                showEmptyGroup={userDisplayFilters?.show_empty_groups ?? true}
                quickAddCallback={createIssue}
                disableIssueCreation={!enableIssueCreation || !isEditingAllowed}
                canEditProperties={canEditProperties}
                scrollableContainerRef={scrollableContainerRef}
                handleOnDrop={handleOnDrop}
                loadMoreIssues={fetchMoreIssues}
              />
            </div>
          </div>
        </div>
      </IssueLayoutHOC>
    </>
  );
});
