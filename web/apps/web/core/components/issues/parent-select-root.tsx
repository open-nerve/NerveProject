/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import { useDetailCommand } from "./issue-detail/use-issue-operations";
// components
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueParentSelect } from "@/components/issues/issue-detail/parent-select";
// hooks
import { useDetailScope } from "./issue-detail-widgets/use-detail-scope";
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";

type TIssueParentSelect = {
  className?: string;
  disabled?: boolean;
  issueId: string;
  issueOperations: TIssueOperations;
  projectId: string;
  workspaceSlug: string;
};

export const IssueParentSelectRoot = observer(function IssueParentSelectRoot(props: TIssueParentSelect) {
  const { issueId, issueOperations, projectId, workspaceSlug } = props;
  const run = useDetailCommand(workspaceSlug, projectId, issueId);
  const capture = useDetailScope(workspaceSlug, projectId, issueId);
  // store hooks
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const {
    removeSubIssue,
    subIssues: { setSubIssueHelpers, subIssueHelpersByIssueId },
  } = useIssueDetail();
  const { toggleParentIssueModal } = useIssueDetailModals();

  // derived values
  const issue = getIssueById(issueId);
  const parentIssue = issue?.parent_id ? getIssueById(issue.parent_id) : undefined;

  const handleParentIssue = async (_issueId: string | null = null) => {
    const current = capture();
    try {
      await issueOperations.update(workspaceSlug, projectId, issueId, { parent_id: _issueId });
      if (!current()) return;
      if (current()) toggleParentIssueModal(null);
    } catch (_error) {
      // The command reported the refusal; retain the selection dialog.
    }
  };

  const handleRemoveSubIssue = async (ws: string, project: string, parent: string, child: string) => {
    if (subIssueHelpersByIssueId(parent).issue_loader.includes(child)) return;
    setSubIssueHelpers(parent, "issue_loader", child);
    try {
      await run(() => removeSubIssue(ws, project, parent, child));
    } catch (_error) {
      // useDetailCommand reports native refusals and suppresses stale-host feedback.
    } finally {
      if (subIssueHelpersByIssueId(parent).issue_loader.includes(child))
        setSubIssueHelpers(parent, "issue_loader", child);
    }
  };

  const workItemLink = `/${workspaceSlug}/projects/${parentIssue?.project_id}/issues/${parentIssue?.id}`;

  if (!issue) return <></>;

  return (
    <IssueParentSelect
      {...props}
      handleParentIssue={handleParentIssue}
      handleRemoveSubIssue={handleRemoveSubIssue}
      workItemLink={workItemLink}
    />
  );
});
