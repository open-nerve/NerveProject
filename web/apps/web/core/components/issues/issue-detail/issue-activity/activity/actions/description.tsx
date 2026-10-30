/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { AlignLeftOutline } from "@makeplane/propel/icons";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { IssueActivityBlockComponent, IssueLink } from "./";

type TIssueDescriptionActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  showIssue?: boolean;
  ends: "top" | "bottom" | undefined;
};

export const IssueDescriptionActivity = observer(function IssueDescriptionActivity(props: TIssueDescriptionActivity) {
  const { activityId, issueId, workspaceSlug, showIssue = true, ends } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<AlignLeftOutline width={14} height={14} className="text-secondary" aria-hidden="true" />}
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      ends={ends}
    >
      <>
        updated the description
        {showIssue ? ` of ` : ``}
        {showIssue && <IssueLink activityId={activityId} issueId={issueId} workspaceSlug={workspaceSlug} />}.
      </>
    </IssueActivityBlockComponent>
  );
});
