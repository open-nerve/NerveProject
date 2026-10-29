/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueUpdate } from "@nerve/api-client";
import type { IssueDrop } from "@/store/issue/root.store";

export const handleDragDrop = async (
  issueId: string,
  sourceDate: string,
  destinationDate: string,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  updateIssue?: (
    projectId: string,
    issueId: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop: IssueDrop
  ) => Promise<void>
) => {
  if (!workspaceSlug || !projectId || !updateIssue) return;

  if (sourceDate === destinationDate) return;

  return await updateIssue(projectId, issueId, () => ({ target_date: destinationDate }), { group: destinationDate });
};
