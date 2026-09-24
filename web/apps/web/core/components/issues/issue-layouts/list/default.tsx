/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
// types
import type {
  GroupByColumnTypes,
  TGroupedIssues,
  TIssue,
  IIssueDisplayProperties,
  TIssueMap,
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
  groupedIssueIds: TGroupedIssues;
  issuesMap: TIssueMap;
  group_by: TIssueGroupByOptions | null;
  orderBy: TIssueOrderByOptions | undefined;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<void>) | undefined;
  quickActions: TRenderQuickActions;
  displayProperties: IIssueDisplayProperties | undefined;
  enableIssueQuickAdd: boolean;
  showEmptyGroup?: boolean;
  canEditProperties: (projectId: string | undefined) => boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
  disableIssueCreation?: boolean;
  handleOnDrop: (source: GroupDropLocation, destination: GroupDropLocation) => Promise<void>;
  addIssuesToView?: (issueIds: string[]) => Promise<TIssue>;
  isCompletedCycle?: boolean;
  loadMoreIssues: (groupId?: string) => void;
  handleCollapsedGroups: (value: string) => void;
  collapsedGroups: TIssueKanbanFilters;
}

export const List = observer(function List(props: IList) {
  const {
    groupedIssueIds,
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
    addIssuesToView,
    isCompletedCycle = false,
    loadMoreIssues,
    handleCollapsedGroups,
    collapsedGroups,
  } = props;

  const storeType = useIssueStoreType();

  const containerRef = useRef<HTMLDivElement | null>(null);

  const groups = getGroupByColumns({
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

  if (!groups) return null;

  const getGroupIndex = (groupId: string | undefined) => groups.findIndex(({ id }) => id === groupId);

  return (
    <div className="relative flex size-full flex-col">
      {groups && (
        <div
          ref={containerRef}
          className="vertical-scrollbar relative scrollbar-lg size-full overflow-auto bg-surface-1"
        >
          {groups.map((group: IGroupByColumn) => (
            <ListGroup
              key={group.id}
              groupIssueIds={groupedIssueIds?.[group.id]}
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
              addIssuesToView={addIssuesToView}
              isCompletedCycle={isCompletedCycle}
              loadMoreIssues={loadMoreIssues}
              containerRef={containerRef}
              handleCollapsedGroups={handleCollapsedGroups}
              collapsedGroups={collapsedGroups}
            />
          ))}
        </div>
      )}
    </div>
  );
});
