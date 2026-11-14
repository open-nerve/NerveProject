/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import type { GroupJoinRefusal, IPragmaticDropPayload, IssueChange, TIssueGroupByOptions } from "@nerve/types";

// store
import type { IssueDrop } from "@/store/issue/issue-writes";

export type GroupDropLocation = {
  groupId: string;
  subGroupId?: string;
  id: string | undefined;
  canAddIssueBelow?: boolean;
};

/**
 * get Kanban Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getSourceFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location, source: sourceIssue } = payload;

  const sourceIssueData = sourceIssue.data;
  let sourceColumData;

  const sourceDropTargets = location?.initial?.dropTargets ?? [];
  for (const dropTarget of sourceDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN") {
      sourceColumData = dropTargetData;
    }
  }

  if (
    typeof sourceIssueData?.id !== "string" ||
    !sourceIssueData.id ||
    typeof sourceColumData?.groupId !== "string" ||
    (sourceColumData.subGroupId !== undefined && typeof sourceColumData.subGroupId !== "string")
  )
    return;

  return {
    groupId: sourceColumData.groupId,
    subGroupId: sourceColumData.subGroupId as string | undefined,
    id: sourceIssueData.id,
  };
};

/**
 * get Destination Source data from Pragmatic Payload
 * @param payload
 * @returns
 */
export const getDestinationFromDropPayload = (payload: IPragmaticDropPayload): GroupDropLocation | undefined => {
  const { location } = payload;

  let destinationIssueData, destinationColumnData;

  const destDropTargets = location?.current?.dropTargets ?? [];

  for (const dropTarget of destDropTargets) {
    const dropTargetData = dropTarget?.data;

    if (!dropTargetData) continue;

    if (dropTargetData.type === "COLUMN" || dropTargetData.type === "DELETE") {
      destinationColumnData = dropTargetData;
    }

    if (dropTargetData.type === "ISSUE") {
      destinationIssueData = dropTargetData;
    }
  }

  if (
    typeof destinationColumnData?.groupId !== "string" ||
    (destinationColumnData.subGroupId !== undefined && typeof destinationColumnData.subGroupId !== "string") ||
    (destinationIssueData && (typeof destinationIssueData.id !== "string" || !destinationIssueData.id))
  )
    return;

  return {
    groupId: destinationColumnData.groupId,
    subGroupId: destinationColumnData.subGroupId as string | undefined,
    id: destinationIssueData?.id as string | undefined,
    canAddIssueBelow: !!destinationIssueData && extractClosestEdge(destinationIssueData) === "bottom",
  };
};

/**
 * Why nerve refuses the drop, which is then not sent (P7b spec 2.4, 2.7; R4): the refusal of the column the work item is
 * dragged into from another (column, the column's field), or of the board's cell's lane it is dragged into from another
 * (lane). A move within a column or a lane joins nothing and changes only the order, which nerve takes.
 */
export const dropRefusal = (
  source: GroupDropLocation,
  destination: GroupDropLocation,
  column: GroupJoinRefusal | undefined,
  lane?: GroupJoinRefusal
): GroupJoinRefusal | undefined =>
  (source.groupId === destination.groupId ? undefined : column) ??
  (source.subGroupId === destination.subGroupId ? undefined : lane);

/**
 * Sends a drop as the change of the target's values (P7b spec 2.7, 2.11; R3), worked out in the write's turn from the
 * work item then held: a cycle as one value, None's null; the modules as the source left and the target joined, applied
 * to the set held then (changeBody), so a module the page does not show, archived too, stays and None leaves the source
 * only; labels and assignees as the set held with the source replaced, None's empty.
 */
export const handleGroupDragDrop = async (
  source: GroupDropLocation,
  destination: GroupDropLocation,
  getIssueById: (id: string) => Issue | undefined,
  getIssueIds: (group?: string, subgroup?: string) => string[] | undefined,
  updateIssueOnDrop: (
    project: string,
    id: string,
    makeChange: (held: Issue) => IssueChange,
    drop: IssueDrop
  ) => Promise<void>,
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy: TIssueGroupByOptions | undefined,
  shouldAddIssueAtTop = false
) => {
  if (!source.id || source.id === destination.id) return false;
  const issue = getIssueById(source.id);
  if (!issue) return false;
  const ids = (getIssueIds(destination.groupId, destination.subGroupId) ?? []).filter((id) => id !== source.id);
  let index = destination.id ? ids.indexOf(destination.id) : shouldAddIssueAtTop ? 0 : ids.length;
  // An invalid captured card is not the same as an intentional column-tail drop.
  if (index < 0) return false;
  if (destination.id && destination.canAddIssueBelow) index++;
  const drop: IssueDrop = {
    group: destination.groupId,
    subgroup: destination.subGroupId,
    beforeId: ids[index],
    afterId: ids[index - 1],
  };
  const makeChange = (held: Issue): IssueChange => {
    const change: IssueChange = {};
    const move = (field: TIssueGroupByOptions | undefined, from: string | undefined, to: string | undefined) => {
      if (!field || to === undefined || from === to) return;
      switch (field) {
        case "priority":
          if (!["none", "low", "medium", "high", "urgent"].includes(to)) throw new Error("Invalid priority drop");
          change.priority = to as IssueChange["priority"];
          break;
        case "state":
          if (!to || to === "none") throw new Error("Invalid state drop");
          change.state_id = to;
          break;
        case "assignees":
        case "labels": {
          const key = field === "assignees" ? "assignee_ids" : "label_ids";
          change[key] = to === "none" ? [] : [...new Set([...held[key].filter((id) => id !== from), to])];
          break;
        }
        case "cycle":
          change.cycle_id = to === "none" ? null : to;
          break;
        case "module":
          change.modules = {
            remove: from === undefined || from === "none" ? [] : [from],
            add: to === "none" ? [] : [to],
          };
          break;
        default:
          throw new Error("Unsupported drop group");
      }
    };
    move(groupBy, source.groupId, destination.groupId);
    move(subGroupBy, source.subGroupId, destination.subGroupId);
    return change;
  };
  await updateIssueOnDrop(issue.project_id, issue.id, makeChange, drop);
  return true;
};
