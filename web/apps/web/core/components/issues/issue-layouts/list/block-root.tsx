/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { MutableRefObject } from "react";
import React, { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/element/adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import { observer } from "mobx-react";
// nerve helpers
import { useOutsideClickDetector } from "@nerve/hooks";
// types
import type { IIssueDisplayProperties } from "@nerve/types";
// components
import { DropIndicator } from "@nerve/ui";
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { ListLoaderItemRow } from "@/components/ui/loader/layouts/list-layout-loader";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { usePlatformOS } from "@/hooks/use-platform-os";
// types
import { HIGHLIGHT_CLASS, getIssueBlockId, isIssueNew } from "../utils";
import { IssueBlock } from "./block";
import { useRegisterIdentifierWidth } from "./identifier-width";
import type { TRenderQuickActions } from "./list-view-types";

type Props = {
  issueId: string;
  issuesMap: Readonly<Record<string, Issue | IssueDetail>>;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  displayProperties: IIssueDisplayProperties | undefined;
  nestingLevel: number;
  spacingLeft?: number;
  containerRef: MutableRefObject<HTMLDivElement | null>;
  groupId: string;
  isDragAllowed: boolean;
  canDropOverIssue: boolean;
  isParentIssueBeingDragged?: boolean;
  isLastChild?: boolean;
  shouldRenderByDefault?: boolean;
};

export const IssueBlockRoot = observer(function IssueBlockRoot(props: Props) {
  const {
    issueId,
    issuesMap,
    groupId,
    updateIssue,
    quickActions,
    canEditProperties,
    displayProperties,
    nestingLevel,
    spacingLeft = 14,
    containerRef,
    isDragAllowed,
    canDropOverIssue,
    isParentIssueBeingDragged = false,
    isLastChild = false,
    shouldRenderByDefault,
  } = props;
  // states
  const [isExpanded, setExpanded] = useState<boolean>(false);
  const [instruction, setInstruction] = useState<"DRAG_OVER" | "DRAG_BELOW" | undefined>(undefined);
  const [isCurrentBlockDragging, setIsCurrentBlockDragging] = useState(false);
  // ref
  const issueBlockRef = useRef<HTMLDivElement | null>(null);
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const { subIssues: subIssuesStore } = useIssueDetail();

  useRegisterIdentifierWidth(issuesMap[issueId]);

  const isSubIssue = nestingLevel !== 0;

  useEffect(() => {
    const blockElement = issueBlockRef.current;

    if (!blockElement) return;

    return combine(
      dropTargetForElements({
        element: blockElement,
        canDrop: ({ source }) => source?.data?.id !== issueId && !isSubIssue && canDropOverIssue,
        getData: ({ input, element }) => {
          const data = { id: issueId, type: "ISSUE" };

          return attachClosestEdge(data, { input, element, allowedEdges: ["top", "bottom"] });
        },
        onDrag: ({ self }) => {
          const edge = extractClosestEdge(self.data);
          setInstruction(edge === "bottom" ? "DRAG_BELOW" : edge === "top" ? "DRAG_OVER" : undefined);
        },
        onDragLeave: () => {
          setInstruction(undefined);
        },
        onDrop: () => {
          setInstruction(undefined);
        },
      })
    );
  }, [issueId, issueBlockRef, isSubIssue, canDropOverIssue, setInstruction]);

  useOutsideClickDetector(issueBlockRef, () => {
    issueBlockRef?.current?.classList?.remove(HIGHLIGHT_CLASS);
  });

  if (!issueId || !issuesMap[issueId]?.created_at) return null;

  const subIssues = subIssuesStore.subIssuesByIssueId(issueId);
  return (
    <div className="relative" ref={issueBlockRef} id={getIssueBlockId(issueId, groupId)}>
      <DropIndicator classNames="absolute top-0 z-[2]" isVisible={instruction === "DRAG_OVER"} />
      <RenderIfVisible
        key={`${issueId}`}
        root={containerRef}
        classNames={`relative ${isLastChild && !isExpanded ? "" : "border-b border-b-subtle"}`}
        verticalOffset={100}
        defaultValue={shouldRenderByDefault || (issuesMap[issueId] ? isIssueNew(issuesMap[issueId]) : false)}
        placeholderChildren={<ListLoaderItemRow shouldAnimate={false} renderForPlaceHolder defaultPropertyCount={4} />}
        shouldRecordHeights={isMobile}
      >
        <IssueBlock
          issueId={issueId}
          issuesMap={issuesMap}
          groupId={groupId}
          updateIssue={updateIssue}
          quickActions={quickActions}
          canEditProperties={canEditProperties}
          displayProperties={displayProperties}
          isExpanded={isExpanded}
          setExpanded={setExpanded}
          nestingLevel={nestingLevel}
          spacingLeft={spacingLeft}
          canDrag={!isSubIssue && isDragAllowed}
          isCurrentBlockDragging={isParentIssueBeingDragged || isCurrentBlockDragging}
          setIsCurrentBlockDragging={setIsCurrentBlockDragging}
        />
      </RenderIfVisible>

      {isExpanded &&
        subIssues?.map((subIssueId) => (
          <IssueBlockRoot
            key={`${subIssueId}`}
            issueId={subIssueId}
            issuesMap={issuesMap}
            updateIssue={updateIssue}
            quickActions={quickActions}
            canEditProperties={canEditProperties}
            displayProperties={displayProperties}
            nestingLevel={nestingLevel + 1}
            spacingLeft={spacingLeft + 12}
            containerRef={containerRef}
            groupId={groupId}
            isDragAllowed={isDragAllowed}
            canDropOverIssue={canDropOverIssue}
            isParentIssueBeingDragged={isParentIssueBeingDragged || isCurrentBlockDragging}
            shouldRenderByDefault={isExpanded}
          />
        ))}
      <DropIndicator classNames="absolute bottom-0 z-[2]" isVisible={instruction === "DRAG_BELOW"} />
    </div>
  );
});
