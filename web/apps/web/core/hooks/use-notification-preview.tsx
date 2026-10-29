/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

export type TNotificationPreview = {
  isWorkItem: boolean;
  PeekOverviewComponent: typeof IssuePeekOverview;
};

/**
 * This function returns if the current active notification is related to a work item.
 * @returns isWorkItem: boolean, peekOverviewComponent: IWorkItemPeekOverview, setPeekWorkItem
 */
export const useNotificationPreview = (): TNotificationPreview => {
  const { peekIssue } = useIssueDetail();

  return {
    isWorkItem: Boolean(peekIssue),
    PeekOverviewComponent: IssuePeekOverview,
  };
};
