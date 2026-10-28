/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

export const handleDragDrop = async (
  issueId: string,
  sourceDate: string,
  destinationDate: string,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  updateIssue?: (projectId: string, issueId: string, data: Partial<IssueDetail>) => Promise<void>
) => {
  if (!workspaceSlug || !projectId || !updateIssue) return;

  if (sourceDate === destinationDate) return;

  const updatedIssue = {
    id: issueId,
    target_date: destinationDate,
  };

  return await updateIssue(projectId, updatedIssue.id, updatedIssue);
};
