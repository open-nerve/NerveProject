/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { observer } from "mobx-react";
import { useSessionSWR } from "@/lib/use-session-swr";
import { DetailResourceError } from "../../issue-detail-widgets/resource-error";

// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// types
import { IssueParentSiblingItem } from "./sibling-item";

export type TIssueParentSiblings = {
  workspaceSlug: string;
  currentIssue: Issue;
  parentId: string;
};

export const IssueParentSiblings = observer(function IssueParentSiblings(props: TIssueParentSiblings) {
  const { workspaceSlug, currentIssue, parentId } = props;
  const projectId = currentIssue.project_id;
  // hooks
  const {
    fetchSubIssues,
    rootIssueStore: root,
    issue: { getIssueById },
    subIssues: { subIssuesByIssueId },
  } = useIssueDetail();

  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [projectId]);
  const canRead =
    root.active &&
    root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined;
  const { isLoading, error, mutate } = useSessionSWR(
    canRead ? ["ISSUE_PARENT_CHILD_ISSUES", identity, workspaceSlug, projectId, parentId] : null,
    (_identity, ws, project, id) => fetchSubIssues(ws, project, id)
  );

  const subIssueIds = getIssueById(parentId) && subIssuesByIssueId(parentId);

  return (
    <div className="my-1">
      {error ? (
        <DetailResourceError
          error={error}
          retry={() => mutate()}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={parentId}
        />
      ) : isLoading ? (
        <div className="flex items-center gap-2 px-1 py-1 text-left text-11 whitespace-nowrap text-secondary">
          Loading
        </div>
      ) : subIssueIds && subIssueIds.length > 0 ? (
        subIssueIds.map(
          (issueId) =>
            currentIssue.id != issueId && (
              <IssueParentSiblingItem key={issueId} workspaceSlug={workspaceSlug} issueId={issueId} />
            )
        )
      ) : (
        <div className="flex items-center gap-2 px-1 py-1 text-left text-11 whitespace-nowrap text-secondary">
          No sibling work items
        </div>
      )}
    </div>
  );
});
