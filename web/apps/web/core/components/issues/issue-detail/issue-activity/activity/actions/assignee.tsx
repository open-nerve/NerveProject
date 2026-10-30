/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// icons
import { MembersOutline } from "@makeplane/propel/icons";
// hooks;
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueAssigneeActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  showIssue?: boolean;
  ends: "top" | "bottom" | undefined;
};

export const IssueAssigneeActivity = observer(function IssueAssigneeActivity(props: TIssueAssigneeActivity) {
  const { activityId, issueId, workspaceSlug, ends, showIssue = true } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);

  if (!activity) return <></>;
  const assigneeId = activity.new_identifier ?? activity.old_identifier;
  return (
    <IssueActivityBlockComponent
      icon={<MembersOutline className="h-3.5 w-3.5 flex-shrink-0 text-secondary" />}
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      ends={ends}
    >
      <>
        {activity.old_value === "" ? `added a new assignee ` : `removed the assignee `}
        <a
          href={assigneeId ? `/${workspaceSlug}/profile/${assigneeId}` : undefined}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center font-medium text-primary capitalize hover:underline"
        >
          {activity.new_value && activity.new_value !== "" ? activity.new_value : activity.old_value}
        </a>
        {showIssue && (activity.old_value === "" ? ` to ` : ` from `)}
        {showIssue && <IssueLink activityId={activityId} issueId={issueId} workspaceSlug={workspaceSlug} />}.
      </>
    </IssueActivityBlockComponent>
  );
});
