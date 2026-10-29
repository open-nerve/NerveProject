/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { followInSession } from "@/lib/in-session";

import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EIssuesStoreType } from "@nerve/types";
import type { TIssueGroupByOptions, TIssueOrderByOptions } from "@nerve/types";
import type { GroupDropLocation } from "@/components/issues/issue-layouts/utils";
import { handleGroupDragDrop } from "@/components/issues/issue-layouts/utils";
import { useIssueDetail } from "./store/use-issue-detail";
import { useIssues } from "./store/use-issues";
import { useIssuesActions } from "./use-issues-actions";

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
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const dropIssue = useIssuesActions(storeType === EIssuesStoreType.WORKSPACE_DRAFT ? undefined : storeType)?.dropIssue;
  const {
    issues: { getIssueIds },
  } = useIssues(storeType);

  const handleOnDrop = async (source: GroupDropLocation, destination: GroupDropLocation) => {
    if (
      source.columnId &&
      destination.columnId &&
      destination.columnId === source.columnId &&
      destination.id === source.id
    )
      return;

    if (!dropIssue) return;
    await followInSession(
      () =>
        handleGroupDragDrop(
          source,
          destination,
          getIssueById,
          getIssueIds,
          dropIssue,
          groupBy,
          subGroupBy,
          orderBy !== "sort_order"
        ),
      {
        failed: () => {
          setToast({
            title: "Error!",
            type: TOAST_TYPE.ERROR,
            message: "Failed to perform this action",
          });
        },
      }
    );
  };

  return handleOnDrop;
};
