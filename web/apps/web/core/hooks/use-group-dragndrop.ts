/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useParams } from "react-router";
import { followInSession, sessionGuard } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { EIssuesStoreType } from "@nerve/types";
import type { TIssueGroupByOptions, TIssueOrderByOptions } from "@nerve/types";
import type { GroupDropLocation } from "@/components/issues/issue-layouts/utils";
import { getIssueBlockId, handleGroupDragDrop } from "@/components/issues/issue-layouts/utils";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useIssueDetail } from "./store/use-issue-detail";
import { useIssues } from "./store/use-issues";
import { useIssuesActions } from "./use-issues-actions";
import { useRefusalToast } from "./use-refusal-toast";
import { useDropHighlight } from "./use-drop-highlight";

type DNDStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.PROFILE
  | EIssuesStoreType.ARCHIVED
  | EIssuesStoreType.WORKSPACE_DRAFT;

export const useGroupIssuesDragNDrop = (
  storeType: DNDStoreType,
  orderBy: TIssueOrderByOptions | undefined,
  groupBy: TIssueGroupByOptions | undefined,
  subGroupBy?: TIssueGroupByOptions
) => {
  const { workspaceSlug, projectId } = useParams();
  const {
    rootIssueStore: root,
    issue: { getIssueById },
  } = useIssueDetail();
  const dropIssue = useIssuesActions(storeType === EIssuesStoreType.WORKSPACE_DRAFT ? undefined : storeType)?.dropIssue;
  const { issues } = useIssues(storeType);
  const scope = JSON.stringify([storeType, orderBy, groupBy, subGroupBy]);
  const capture = useDetailScope(workspaceSlug ?? "", projectId, scope, true);
  const highlight = useDropHighlight(JSON.stringify([workspaceSlug, projectId, scope]));
  const refuse = useRefusalToast();

  return async (source: GroupDropLocation, destination: GroupDropLocation) => {
    if (!dropIssue || !source.id || source.id === destination.id) return;
    // Other sorts may move between groups, but never manually reorder within one group.
    if (
      orderBy !== "sort_order" &&
      source.groupId === destination.groupId &&
      source.subGroupId === destination.subGroupId
    )
      return;
    const issue = getIssueById(source.id);
    if (!issue || !("currentList" in issues)) return;
    const binding = issues.currentList;
    const sameHost = capture();
    const sameSession = sessionGuard();
    const permission = root.permissionReads.identity(issue.workspace_id, [issue.project_id], true);
    const current = () =>
      sameHost() &&
      sameSession() &&
      issues.currentList === binding &&
      root.permissionReads.identity(issue.workspace_id, [issue.project_id], true) === permission;
    if (!binding || !current()) return;
    await followInSession(
      () =>
        handleGroupDragDrop(
          source,
          destination,
          getIssueById,
          issues.getIssueIds,
          dropIssue,
          groupBy,
          subGroupBy,
          orderBy !== "sort_order"
        ),
      {
        done: (changed) => {
          if (changed && current())
            highlight(
              getIssueBlockId(source.id, destination.groupId, destination.subGroupId),
              orderBy !== "sort_order",
              current
            );
        },
        failed: (error) => {
          if (current() && !(error instanceof SessionChangedError)) refuse(error);
        },
      }
    );
  };
};
