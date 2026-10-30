import { useParams } from "react-router";
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
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IIssueDisplayProperties, TIssueGroupByOptions, TIssueOrderByOptions } from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
import { cn } from "@nerve/utils";
import type { GroupDropLocation } from "@/components/issues/issue-layouts/utils";
// components
import { getSourceFromDropPayload, getDestinationFromDropPayload } from "@/components/issues/issue-layouts/utils";
import { KanbanIssueBlockLoader } from "@/components/ui/loader/layouts/kanban-layout-loader";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useIssuesStore } from "@/hooks/use-issue-layout-store";
// local imports
import { GroupDragOverlay } from "../group-drag-overlay";
import type { TRenderQuickActions } from "../list/list-view-types";
import { KanbanQuickAddIssueButton, QuickAddIssueRoot } from "../quick-add";
import { KanbanIssueBlocksList } from "./blocks-list";

interface IKanbanGroup {
  groupId: string;
  issuesMap: Readonly<Record<string, Issue | IssueDetail>>;
  groups: Map<string, Group>;
  displayProperties: IIssueDisplayProperties | undefined;
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  sub_group_id?: string;
  isDragDisabled: boolean;
  isDropDisabled: boolean;
  dropErrorMessage: string | undefined;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
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
    groupId,
    sub_group_id,
    group_by,
    orderBy,
    sub_group_by,
    issuesMap,
    displayProperties,
    groups,
    isDropDisabled,
    isDragDisabled,
    dropErrorMessage,
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
  const projectState = useProjectState();
  const { projectId } = useParams();

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
        getData: () => ({ groupId, subGroupId: sub_group_id, columnId: `${groupId}__${sub_group_id}`, type: "COLUMN" }),
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

          if (isDropDisabled) {
            if (dropErrorMessage)
              setToast({
                type: TOAST_TYPE.WARNING,
                title: t("common.warning"),
                message: dropErrorMessage,
              });
            return;
          }

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
    isDropDisabled,
    dropErrorMessage,
    handleOnDrop,
    t,
  ]);

  const prePopulateQuickAddData = (
    groupByKey: string | undefined,
    subGroupByKey: string | undefined | null,
    groupValue: string,
    subGroupValue: string | undefined
  ) => {
    const defaultState = projectState.getProjectStates(projectId)?.find((state) => state.default);
    let preloadedData: object = { state_id: defaultState?.id };

    if (groupByKey) {
      if (groupByKey === "state") {
        preloadedData = { ...preloadedData, state_id: groupValue };
      } else if (groupByKey === "priority") {
        preloadedData = { ...preloadedData, priority: groupValue };
      } else if (groupByKey === "labels") {
        if (groupValue !== "none") preloadedData = { ...preloadedData, label_ids: [groupValue] };
      } else if (groupByKey === "assignees") {
        if (groupValue !== "none") preloadedData = { ...preloadedData, assignee_ids: [groupValue] };
      } else if (groupByKey === "created_by") {
        preloadedData = { ...preloadedData };
      } else {
        preloadedData = { ...preloadedData, [groupByKey]: groupValue };
      }
    }

    if (subGroupByKey && subGroupValue !== undefined) {
      if (subGroupByKey === "state") {
        preloadedData = { ...preloadedData, state_id: subGroupValue };
      } else if (subGroupByKey === "priority") {
        preloadedData = { ...preloadedData, priority: subGroupValue };
      } else if (subGroupByKey === "labels") {
        if (subGroupValue !== "none") preloadedData = { ...preloadedData, label_ids: [subGroupValue] };
      } else if (subGroupByKey === "assignees") {
        if (subGroupValue !== "none") preloadedData = { ...preloadedData, assignee_ids: [subGroupValue] };
      } else if (subGroupByKey === "created_by") {
        preloadedData = { ...preloadedData };
      } else {
        preloadedData = { ...preloadedData, [subGroupByKey]: subGroupValue };
      }
    }

    return preloadedData;
  };

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
  const canOverlayBeVisible = orderBy !== "sort_order" || isDropDisabled;
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
        isDropDisabled={isDropDisabled}
        dropErrorMessage={dropErrorMessage}
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

      {enableQuickIssueCreate && !disableIssueCreation && (
        <div className="sticky bottom-0 w-full bg-surface-2 py-0.5">
          <QuickAddIssueRoot
            scopeKey={JSON.stringify([group_by, groupId, sub_group_by, sub_group_id])}
            layout={EIssueLayoutTypes.KANBAN}
            QuickAddButton={KanbanQuickAddIssueButton}
            prePopulatedData={{
              ...(group_by && prePopulateQuickAddData(group_by, sub_group_by, groupId, sub_group_id)),
            }}
            quickAddCallback={quickAddCallback}
          />
        </div>
      )}
    </div>
  );
});
