import type { IssueCreate } from "@nerve/api-client";
import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import { useEffect, useRef } from "react";
import { useParams } from "react-router";
import { useSession } from "@/lib/auth/use-session";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { IdentifierWidthScope } from "./identifier-width";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
// types
import type {
  GroupByColumnTypes,
  IIssueDisplayProperties,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
  IGroupByColumn,
  TIssueKanbanFilters,
} from "@nerve/types";
// hooks
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
// utils
import type { GroupDropLocation } from "../utils";
import { getGroupByColumns, isWorkspaceLevel } from "../utils";
import { ListGroup } from "./list-group";
import type { TRenderQuickActions } from "./list-view-types";

export interface IList {
  viewId?: string;
  groups: Map<string, Group>;
  issuesMap: Readonly<Record<string, Issue | IssueDetail>>;
  group_by: TIssueGroupByOptions | null;
  orderBy: TIssueOrderByOptions | undefined;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
  quickActions: TRenderQuickActions;
  displayProperties: IIssueDisplayProperties | undefined;
  enableIssueQuickAdd: boolean;
  showEmptyGroup?: boolean;
  canEditProperties: (projectId: string | undefined) => boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  disableIssueCreation?: boolean;
  handleOnDrop: (source: GroupDropLocation, destination: GroupDropLocation) => Promise<void>;
  loadMoreIssues: (groupId?: string) => void;
  handleCollapsedGroups: (value: string) => void;
  collapsedGroups: TIssueKanbanFilters;
}

export const List = observer(function List(props: IList) {
  const {
    groups,
    issuesMap,
    group_by,
    orderBy,
    updateIssue,
    quickActions,
    displayProperties,
    enableIssueQuickAdd,
    showEmptyGroup,
    canEditProperties,
    quickAddCallback,
    disableIssueCreation,
    handleOnDrop,
    loadMoreIssues,
    handleCollapsedGroups,
    collapsedGroups,
  } = props;

  const storeType = useIssueStoreType();
  const { loginId } = useSession();
  const { workspaceSlug, projectId, userId } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(workspace?.id, projectId ? [projectId] : []);
  const scope = JSON.stringify([loginId, workspaceSlug, storeType, projectId, userId, props.viewId, identity]);

  const containerRef = useRef<HTMLDivElement | null>(null);

  const columns = getGroupByColumns({
    groupBy: group_by as GroupByColumnTypes,
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
  });

  // Enable Auto Scroll for Main Kanban
  useEffect(() => {
    const element = containerRef.current;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, [containerRef]);

  if (!columns) return null;

  const getGroupIndex = (groupId: string | undefined) => columns.findIndex(({ id }) => id === groupId);

  return (
    <div className="relative flex size-full flex-col">
      {columns && (
        <div
          ref={containerRef}
          className="vertical-scrollbar relative scrollbar-lg size-full overflow-auto bg-surface-1"
        >
          <IdentifierWidthScope key={scope} owner={root}>
            {columns.map((group: IGroupByColumn) => (
              <ListGroup
                key={group.id}
                groupIssueIds={groups?.get(group.id)?.rows.map((row) => row.id)}
                issuesMap={issuesMap}
                group_by={group_by}
                group={group}
                updateIssue={updateIssue}
                quickActions={quickActions}
                orderBy={orderBy}
                getGroupIndex={getGroupIndex}
                handleOnDrop={handleOnDrop}
                displayProperties={displayProperties}
                enableIssueQuickAdd={enableIssueQuickAdd}
                showEmptyGroup={showEmptyGroup}
                canEditProperties={canEditProperties}
                quickAddCallback={quickAddCallback}
                disableIssueCreation={disableIssueCreation}
                loadMoreIssues={loadMoreIssues}
                containerRef={containerRef}
                handleCollapsedGroups={handleCollapsedGroups}
                collapsedGroups={collapsedGroups}
              />
            ))}
          </IdentifierWidthScope>
        </div>
      )}
    </div>
  );
});
