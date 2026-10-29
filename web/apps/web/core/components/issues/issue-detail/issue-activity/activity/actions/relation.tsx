/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// types
import type { TIssueRelationTypes } from "@nerve/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { ISSUE_RELATION_OPTIONS } from "@/components/relations";
// local helpers
import { IssueActivityBlockComponent } from "./";
import { getRelationActivityContent } from "./helpers/activity";

type TIssueRelationActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  ends: "top" | "bottom" | undefined;
};

export const IssueRelationActivity = observer(function IssueRelationActivity(props: TIssueRelationActivity) {
  const { activityId, issueId, workspaceSlug, ends } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);
  const activityContent = getRelationActivityContent(activity);

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={activity.field ? ISSUE_RELATION_OPTIONS[activity.field as TIssueRelationTypes]?.icon(14) : <></>}
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      ends={ends}
    >
      {activityContent}
      {activity.old_value === "" ? (
        <span className="font-medium text-primary">{activity.new_value}.</span>
      ) : (
        <span className="font-medium text-primary">{activity.old_value}.</span>
      )}
    </IssueActivityBlockComponent>
  );
});
