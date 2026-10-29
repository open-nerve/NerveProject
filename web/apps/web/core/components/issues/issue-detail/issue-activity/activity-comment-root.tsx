/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { E_SORT_ORDER, TActivityFilters, EActivityFilterType } from "@nerve/constants";
import { BASE_ACTIVITY_FILTER_TYPES, filterActivityOnSelectedFilters } from "@nerve/constants";
import type { TCommentsOperations } from "@nerve/types";
// components
import { CommentCard } from "@/components/comments/card/root";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// local imports
import { IssueActivityItem } from "./activity/activity-list";
import { IssueActivityLoader } from "./loader";
import { DetailResourceError } from "../../issue-detail-widgets/resource-error";

type TIssueActivityCommentRoot = {
  workspaceSlug: string;
  projectId: string;
  isIntakeIssue: boolean;
  issueId: string;
  selectedFilters: TActivityFilters[];
  activityOperations: TCommentsOperations;
  disabled?: boolean;
  sortOrder: E_SORT_ORDER;
};

export const IssueActivityCommentRoot = observer(function IssueActivityCommentRoot(props: TIssueActivityCommentRoot) {
  const { workspaceSlug, isIntakeIssue, issueId, selectedFilters, activityOperations, projectId, disabled, sortOrder } =
    props;
  // store hooks
  const {
    activity: { getActivityAndCommentsByIssueId },
    comment: { getCommentById, getError, fetchComments },
  } = useIssueDetail();
  // derived values
  const activityAndComments = getActivityAndCommentsByIssueId(issueId, sortOrder);
  const error = getError(issueId);
  const readError = (
    <DetailResourceError
      error={error}
      retry={() => fetchComments(workspaceSlug, projectId, issueId)}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
    />
  );

  if (!activityAndComments) return error ? readError : <IssueActivityLoader />;

  if (activityAndComments.length <= 0) return readError;

  const filteredActivityAndComments = filterActivityOnSelectedFilters(activityAndComments, selectedFilters);

  return (
    <div>
      {readError}
      {filteredActivityAndComments.map((activityComment, index) => {
        const comment = getCommentById(issueId, activityComment.id);
        return activityComment.activity_type === "COMMENT" ? (
          <CommentCard
            key={activityComment.id}
            workspaceSlug={workspaceSlug}
            entityId={issueId}
            comment={comment}
            activityOperations={activityOperations}
            ends={index === 0 ? "top" : index === filteredActivityAndComments.length - 1 ? "bottom" : undefined}
            showCopyLinkOption={!isIntakeIssue}
            disabled={disabled}
            projectId={projectId}
            enableReplies
          />
        ) : BASE_ACTIVITY_FILTER_TYPES.includes(activityComment.activity_type as EActivityFilterType) ? (
          <IssueActivityItem
            key={activityComment.id}
            activityId={activityComment.id}
            ends={index === 0 ? "top" : index === filteredActivityAndComments.length - 1 ? "bottom" : undefined}
          />
        ) : null;
      })}
    </div>
  );
});
