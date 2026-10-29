/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// hooks
import { ModuleOutline } from "@makeplane/propel/icons";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { IssueActivityBlockComponent } from "./";

type TIssueModuleActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  ends: "top" | "bottom" | undefined;
};

export const IssueModuleActivity = observer(function IssueModuleActivity(props: TIssueModuleActivity) {
  const { activityId, issueId, workspaceSlug, ends } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<ModuleOutline className="h-4 w-4 flex-shrink-0 text-secondary" />}
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      ends={ends}
    >
      <>
        {activity.verb === "created" ? (
          <>
            <span>added this work item to the module </span>
            <a
              href={
                activity.new_identifier
                  ? `/${workspaceSlug}/projects/${activity.project_id}/modules/${activity.new_identifier}`
                  : undefined
              }
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate">{activity.new_value}</span>
            </a>
          </>
        ) : activity.verb === "updated" ? (
          <>
            <span>set the module to </span>
            <a
              href={
                activity.new_identifier
                  ? `/${workspaceSlug}/projects/${activity.project_id}/modules/${activity.new_identifier}`
                  : undefined
              }
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate"> {activity.new_value}</span>
            </a>
          </>
        ) : (
          <>
            <span>removed the work item from the module </span>
            <a
              href={
                activity.old_identifier
                  ? `/${workspaceSlug}/projects/${activity.project_id}/modules/${activity.old_identifier}`
                  : undefined
              }
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 truncate font-medium text-primary hover:underline"
            >
              <span className="truncate"> {activity.old_value}</span>
            </a>
          </>
        )}
      </>
    </IssueActivityBlockComponent>
  );
});
