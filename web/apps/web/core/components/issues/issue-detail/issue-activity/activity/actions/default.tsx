/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { WorkItemsOutline } from "@makeplane/propel/icons";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// local imports
import { IssueActivityBlockComponent } from "./";

type TIssueDefaultActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  ends: "top" | "bottom" | undefined;
};

export const IssueDefaultActivity = observer(function IssueDefaultActivity(props: TIssueDefaultActivity) {
  const { activityId, issueId, workspaceSlug, ends } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);

  if (!activity) return <></>;

  return (
    <IssueActivityBlockComponent
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      icon={<WorkItemsOutline width={14} height={14} className="text-secondary" aria-hidden="true" />}
      ends={ends}
    >
      <>{activity.verb === "created" ? <span> created the work item.</span> : <span> deleted a work item.</span>}</>
    </IssueActivityBlockComponent>
  );
});
