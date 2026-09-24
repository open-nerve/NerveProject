/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { TIssue } from "@nerve/types";
import { IssueDetailRoot } from "@/components/issues/issue-detail/root";

export type TWorkItemDetailRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issue: TIssue | undefined;
};

export const WorkItemDetailRoot = observer(function WorkItemDetailRoot(props: TWorkItemDetailRoot) {
  const { workspaceSlug, projectId, issueId, issue } = props;

  return (
    <IssueDetailRoot
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
      is_archived={!!issue?.archived_at}
    />
  );
});
