/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useDetailCommand } from "./issue-detail/use-issue-operations";
// components
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueParentSelect } from "@/components/issues/issue-detail/parent-select";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";

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
  const removing = useRef(false);
  // store hooks
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { removeSubIssue } = useIssueDetail();
  const { getProjectIdentifierById } = useProject();

  // derived values
  const issue = getIssueById(issueId);
  const parentIssue = issue?.parent_id ? getIssueById(issue.parent_id) : undefined;

  const handleParentIssue = async (_issueId: string | null = null) => {
    await issueOperations.update(workspaceSlug, projectId, issueId, { parent_id: _issueId });
  };

  const handleRemoveSubIssue = async (ws: string, project: string, parent: string, child: string) => {
    if (removing.current) return;
    removing.current = true;
    try {
      await run(() => removeSubIssue(ws, project, parent, child));
    } catch {
      // useDetailCommand reports native refusals and suppresses stale-host feedback.
    } finally {
      removing.current = false;
    }
  };

  const parentIdentifier = parentIssue ? getProjectIdentifierById(parentIssue.project_id) : undefined;
  const workItemLink =
    parentIssue && parentIdentifier
      ? `/${workspaceSlug}/browse/${parentIdentifier}-${parentIssue.sequence_id}`
      : undefined;

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
