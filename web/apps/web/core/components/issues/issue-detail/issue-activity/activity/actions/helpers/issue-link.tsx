/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { observer } from "mobx-react";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { generateWorkItemLink } from "@nerve/utils";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { usePlatformOS } from "@/hooks/use-platform-os";

type IssueLinkProps = { workspaceSlug: string; issueId: string; activityId: string };
export const IssueLink = observer(function IssueLink({ workspaceSlug, issueId, activityId }: IssueLinkProps) {
  const detail = useIssueDetail();
  const { getProjectById } = useProject();
  const { isMobile } = usePlatformOS();
  const activity = detail.activity.getActivityById(issueId, activityId);
  if (!activity) return null;
  const issue = detail.rootIssueStore.issues.getIssueById(issueId);
  const project = getProjectById(activity.project_id);
  if (!issue || !project) return <span>Work items</span>;
  const href = generateWorkItemLink({
    workspaceSlug,
    projectId: activity.project_id,
    issueId,
    projectIdentifier: project.identifier,
    sequenceId: issue.sequence_id,
  });
  return (
    <Tooltip label={issue.name} layout="stacked" disabled={isMobile}>
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1 font-medium text-primary hover:underline"
      >
        {project.identifier}-{issue.sequence_id} <span className="font-regular">{issue.name}</span>
      </a>
    </Tooltip>
  );
});
