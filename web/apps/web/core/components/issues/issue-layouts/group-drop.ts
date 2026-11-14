/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueUpdate } from "@nerve/api-client";

import { extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import type { IPragmaticDropPayload, TIssueGroupByOptions } from "@nerve/types";

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

export const handleGroupDragDrop = async (
  source: GroupDropLocation,
  destination: GroupDropLocation,
  getIssueById: (id: string) => Issue | undefined,
  getIssueIds: (group?: string, subgroup?: string) => string[] | undefined,
  updateIssueOnDrop: (
    project: string,
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
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
  const makeBody = (held: Issue): IssueUpdate => {
    const body: IssueUpdate = {};
    const move = (field: TIssueGroupByOptions | undefined, from: string | undefined, to: string | undefined) => {
      if (!field || to === undefined || from === to) return;
      switch (field) {
        case "priority":
          if (!["none", "low", "medium", "high", "urgent"].includes(to)) throw new Error("Invalid priority drop");
          body.priority = to as IssueUpdate["priority"];
          break;
        case "state":
          if (!to || to === "none") throw new Error("Invalid state drop");
          body.state_id = to;
          break;
        case "assignees":
        case "labels": {
          const key = field === "assignees" ? "assignee_ids" : "label_ids";
          body[key] = to === "none" ? [] : [...new Set([...held[key].filter((id) => id !== from), to])];
          break;
        }
        default:
          throw new Error("Unsupported drop group");
      }
    };
    move(groupBy, source.groupId, destination.groupId);
    move(subGroupBy, source.subGroupId, destination.subGroupId);
    return body;
  };
  await updateIssueOnDrop(issue.project_id, issue.id, makeBody, drop);
  return true;
};
