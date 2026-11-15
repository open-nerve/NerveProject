import type { IssueCreate } from "@nerve/api-client";
import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { MutableRefObject } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/element/adapter";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
// nerve constants
import { DRAG_ALLOWED_GROUPS } from "@nerve/constants";
// i18n
import { useTranslation } from "@nerve/i18n";
//types
import type {
  IGroupByColumn,
  IIssueDisplayProperties,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
  IssueChange,
} from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
import { cn } from "@nerve/utils";
import type { GroupDropLocation } from "@/components/issues/issue-layouts/group-drop";
// components
import {
  dropRefusal,
  getSourceFromDropPayload,
  getDestinationFromDropPayload,
} from "@/components/issues/issue-layouts/group-drop";
import { KanbanIssueBlockLoader } from "@/components/ui/loader/layouts/kanban-layout-loader";
// hooks
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useIssuesStore } from "@/hooks/use-issue-layout-store";
import { useJoinRefusalWarning } from "@/hooks/use-refusal-toast";
// local imports
import { GroupDragOverlay } from "../group-drag-overlay";
import type { TRenderQuickActions } from "../list/list-view-types";
import { KanbanQuickAddIssueButton, QuickAddIssueRoot, useGroupQuickAddPrefill } from "../quick-add";
import { KanbanIssueBlocksList } from "./blocks-list";

interface IKanbanGroup {
  column: IGroupByColumn;
  /** The column of the sub-grouping whose lane the board is in, of a board's lanes. */
  lane: IGroupByColumn | undefined;
  issuesMap: Readonly<Record<string, Issue | IssueDetail>>;
  groups: Map<string, Group>;
  displayProperties: IIssueDisplayProperties | undefined;
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  sub_group_id?: string;
  isDragDisabled: boolean;
  updateIssue: ((projectId: string | null, issueId: string, data: IssueChange) => Promise<void>) | undefined;
  quickActions: TRenderQuickActions;
  enableQuickIssueCreate?: boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  loadMoreIssues: (groupId?: string, subGroupId?: string) => void;
  disableIssueCreation?: boolean;
  canEditProperties: (projectId: string | undefined) => boolean;
  scrollableContainerRef?: MutableRefObject<HTMLDivElement | null>;
  handleOnDrop: (source: GroupDropLocation, destination: GroupDropLocation) => Promise<void>;
  orderBy: TIssueOrderByOptions | undefined;
}

export const KanbanGroup = observer(function KanbanGroup(props: IKanbanGroup) {
  const {
    column,
    lane,
    sub_group_id,
    group_by,
    orderBy,
    sub_group_by,
    issuesMap,
    displayProperties,
    groups,
    isDragDisabled,
    updateIssue,
    quickActions,
    canEditProperties,
    loadMoreIssues,
    enableQuickIssueCreate,
    disableIssueCreation,
    quickAddCallback,
    scrollableContainerRef,
    handleOnDrop,
  } = props;
  // i18n
  const { t } = useTranslation();
  // hooks
  const groupId = column.id;
  const prefill = useGroupQuickAddPrefill(column, lane);
  const columnRefusal = column.refusal;
  const laneRefusal = lane?.refusal;
  // A cell nerve refuses a work item's joining, its column's or its lane's, offers no new work item (P7b R4).
  const refusal = columnRefusal ?? laneRefusal;
  const warnRefusal = useJoinRefusalWarning();

  const { issues } = useIssuesStore();
  const { getGroupIssueCount, getPaginationData, getIssueLoader } = issues;
  const failed = "currentList" in issues && !!issues.currentList?.error;

  const [intersectionElement, setIntersectionElement] = useState<HTMLSpanElement | null>(null);
  const columnRef = useRef<HTMLDivElement | null>(null);

  const containerRef = sub_group_by && scrollableContainerRef ? scrollableContainerRef : columnRef;

  const loadMoreIssuesInThisGroup = useCallback(() => {
    loadMoreIssues(groupId, sub_group_id);
  }, [loadMoreIssues, groupId, sub_group_id]);

  const isPaginating = !!getIssueLoader(groupId, sub_group_id);

  useIntersectionObserver(
    containerRef,
    isPaginating || failed ? null : intersectionElement,
    loadMoreIssuesInThisGroup,
    `0% 100% 100% 100%`
  );
  const [isDraggingOverColumn, setIsDraggingOverColumn] = useState(false);

  // Enable Kanban Columns as Drop Targets
  useEffect(() => {
    const element = columnRef.current;

    if (!element) return;

    return combine(
      dropTargetForElements({
        element,
        getData: () => ({ groupId, subGroupId: sub_group_id, type: "COLUMN" }),
        onDragEnter: () => {
          setIsDraggingOverColumn(true);
        },
        onDragLeave: () => {
          setIsDraggingOverColumn(false);
        },
        onDragStart: () => {
          setIsDraggingOverColumn(true);
        },
        onDrop: (payload) => {
          setIsDraggingOverColumn(false);
          const source = getSourceFromDropPayload(payload);
          const destination = getDestinationFromDropPayload(payload);

          if (!source || !destination) return;
          // A drop into its column or its lane from another nerve refuses is not sent; the page says nerve's sentence.
          const dropRefused = dropRefusal(source, destination, columnRefusal, laneRefusal);
          if (dropRefused) return warnRefusal(dropRefused);

          handleOnDrop(source, destination);
        },
      }),
      autoScrollForElements({
        element,
      })
    );
  }, [
    columnRef,
    groupId,
    sub_group_id,
    setIsDraggingOverColumn,
    orderBy,
    handleOnDrop,
    columnRefusal,
    laneRefusal,
    warnRefusal,
    t,
  ]);

  const isSubGroup = sub_group_id !== undefined;

  const issueIds = isSubGroup
    ? (groups
        .get(groupId)
        ?.subgroups?.get(sub_group_id)
        ?.rows.map((row) => row.id) ?? [])
    : (groups.get(groupId)?.rows.map((row) => row.id) ?? []);

  const groupIssueCount = getGroupIssueCount(groupId, sub_group_id, false) ?? 0;

  const nextCursor = getPaginationData(groupId, sub_group_id)?.nextCursor;

  const loadMore = (
    <button
      type="button"
      disabled={isPaginating}
      aria-busy={isPaginating}
      className="sticky bottom-0 w-full cursor-pointer p-3 text-13 font-medium text-accent-primary hover:text-accent-secondary hover:underline disabled:cursor-wait"
      onClick={loadMoreIssuesInThisGroup}
    >
      {t("common.load_more")} &darr;
    </button>
  );

  const shouldLoadMore = nextCursor === undefined ? issueIds?.length < groupIssueCount : !!nextCursor;
  const canOverlayBeVisible = orderBy !== "sort_order";
  const shouldOverlayBeVisible = isDraggingOverColumn && canOverlayBeVisible;
  const canDragIssuesInCurrentGrouping =
    !isDragDisabled &&
    !!group_by &&
    DRAG_ALLOWED_GROUPS.includes(group_by) &&
    (sub_group_by ? DRAG_ALLOWED_GROUPS.includes(sub_group_by) : true);

  return (
    <div
      id={`${groupId}__${sub_group_id}`}
      className={cn(
        "relative h-full min-h-[120px] transition-all",
        { "rounded-sm bg-layer-1": isDraggingOverColumn },
        { "vertical-scrollbar scrollbar-md": !sub_group_by && !shouldOverlayBeVisible }
      )}
      ref={columnRef}
    >
      <GroupDragOverlay
        dragColumnOrientation={sub_group_by ? "justify-start" : "justify-center"}
        canOverlayBeVisible={canOverlayBeVisible}
        orderBy={orderBy}
        isDraggingOverColumn={isDraggingOverColumn}
      />
      <KanbanIssueBlocksList
        sub_group_id={sub_group_id}
        groupId={groupId}
        issuesMap={issuesMap}
        issueIds={issueIds || []}
        displayProperties={displayProperties}
        updateIssue={updateIssue}
        quickActions={quickActions}
        canEditProperties={canEditProperties}
        scrollableContainerRef={scrollableContainerRef}
        canDropOverIssue={!canOverlayBeVisible}
        canDragIssuesInCurrentGrouping={canDragIssuesInCurrentGrouping}
      />

      {shouldLoadMore && (
        <>
          {loadMore}
          {!isSubGroup && !failed && <KanbanIssueBlockLoader ref={setIntersectionElement} />}
        </>
      )}

      {enableQuickIssueCreate && !disableIssueCreation && !refusal && (
        <div className="sticky bottom-0 w-full bg-surface-2 py-0.5">
          <QuickAddIssueRoot
            scopeKey={JSON.stringify([group_by, groupId, sub_group_by, sub_group_id])}
            layout={EIssueLayoutTypes.KANBAN}
            QuickAddButton={KanbanQuickAddIssueButton}
            prePopulatedData={prefill}
            quickAddCallback={quickAddCallback}
          />
        </div>
      )}
    </div>
  );
});
