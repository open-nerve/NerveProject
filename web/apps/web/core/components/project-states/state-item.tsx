/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/element/adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import { observer } from "mobx-react";
// Nerve
import type { TDraggableData } from "@nerve/constants";
import type { State, StateGroup } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { DropIndicator } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { StateItemTitle, StateUpdate } from "@/components/project-states";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// helpers
type TStateItem = {
  groupKey: StateGroup;
  groupedStates: Record<string, State[]>;
  totalStates: number;
  state: State;
  stateOperationsCallbacks: TStateOperationsCallbacks;
  disabled?: boolean;
  stateItemClassName?: string;
};

export const StateItem = observer(function StateItem(props: TStateItem) {
  const {
    groupKey,
    groupedStates,
    totalStates,
    state,
    stateOperationsCallbacks,
    disabled = false,
    stateItemClassName,
  } = props;
  const toastRefusal = useRefusalToast();
  // ref
  const draggableElementRef = useRef<HTMLDivElement | null>(null);
  // states
  const [updateStateModal, setUpdateStateModal] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [isDraggedOver, setIsDraggedOver] = useState(false);
  const [closestEdge, setClosestEdge] = useState<string | null>(null);
  // derived values
  const isDraggable = totalStates !== 1;
  const commonStateItemListProps = {
    stateCount: totalStates,
    state: state,
    setUpdateStateModal: setUpdateStateModal,
  };

  const handleStateSequence = useCallback(
    async (stateId: string, group: StateGroup, droppedOnId: string, after: boolean) => {
      try {
        await stateOperationsCallbacks.moveState(stateId, group, droppedOnId, after);
      } catch (error) {
        toastRefusal(error);
      }
    },
    [stateOperationsCallbacks, toastRefusal]
  );

  useEffect(() => {
    const elementRef = draggableElementRef.current;
    const initialData: TDraggableData = { groupKey: groupKey, id: state.id };

    if (!elementRef) return;

    // the cleanup: each run registers the element again, with the handlers of this render
    return combine(
      draggable({
        element: elementRef,
        getInitialData: () => initialData,
        onDragStart: () => setIsDragging(true),
        onDrop: () => setIsDragging(false),
        canDrag: () => isDraggable && !disabled,
      }),
      dropTargetForElements({
        element: elementRef,
        getData: ({ input, element }) =>
          attachClosestEdge(initialData, {
            input,
            element,
            allowedEdges: ["top", "bottom"],
          }),
        onDragEnter: (args) => {
          setIsDraggedOver(true);
          setClosestEdge(extractClosestEdge(args.self.data));
        },
        onDragLeave: () => {
          setIsDraggedOver(false);
          setClosestEdge(null);
        },
        onDrop: (data) => {
          setIsDraggedOver(false);
          const { self, source } = data;
          const sourceData = source.data as TDraggableData;
          const destinationData = self.data as TDraggableData;

          if (sourceData && destinationData && sourceData.id) {
            const destinationGroupKey = destinationData.groupKey;
            const edge = extractClosestEdge(destinationData) || undefined;
            handleStateSequence(sourceData.id, destinationGroupKey, destinationData.id, edge === "bottom");
          }
        },
      })
    );
  }, [draggableElementRef, state, groupKey, isDraggable, groupedStates, handleStateSequence, disabled]);
  // DND ends

  if (updateStateModal)
    return (
      <StateUpdate
        state={state}
        updateStateCallback={stateOperationsCallbacks.updateState}
        handleClose={() => setUpdateStateModal(false)}
      />
    );

  return (
    <Fragment>
      {/* draggable drop top indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "top"} />
      <div
        ref={draggableElementRef}
        className={cn(
          "group relative rounded-sm border border-subtle bg-surface-1 px-3.5 py-3",
          isDragging ? `opacity-50` : `opacity-100`,
          totalStates === 1 ? `cursor-auto` : `cursor-grab`,
          stateItemClassName
        )}
      >
        {disabled ? (
          <StateItemTitle {...commonStateItemListProps} disabled />
        ) : (
          <StateItemTitle
            {...commonStateItemListProps}
            disabled={false}
            stateOperationsCallbacks={{
              markStateAsDefault: stateOperationsCallbacks.markStateAsDefault,
              deleteState: stateOperationsCallbacks.deleteState,
            }}
          />
        )}
      </div>
      {/* draggable drop bottom indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "bottom"} />
    </Fragment>
  );
});
