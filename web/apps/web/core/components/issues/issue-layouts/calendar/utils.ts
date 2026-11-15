/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";
import type { IssueDrop } from "@/store/issue/issue-writes";

export const handleDragDrop = async (
  issueId: string,
  sourceDate: string,
  destinationDate: string,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  updateIssue?: (
    projectId: string,
    issueId: string,
    makeChange: (held: Issue) => IssueChange,
    drop: IssueDrop
  ) => Promise<void>
) => {
  if (!workspaceSlug || !projectId || !updateIssue) return;

  if (sourceDate === destinationDate) return;

  return await updateIssue(projectId, issueId, () => ({ target_date: destinationDate }), { group: destinationDate });
};
