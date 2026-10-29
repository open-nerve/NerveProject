/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import type { IssueUpdate } from "@nerve/api-client";
import { copyUrlToClipboard } from "@nerve/utils";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "../use-detail-scope";

export type TRelationIssueOperations = {
  copyLink: (path: string) => void;
  update: (workspaceSlug: string, projectId: string, issueId: string, body: IssueUpdate) => Promise<void>;
  remove: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
};
export function useRelationOperations(workspaceSlug: string, hostId: string): TRelationIssueOperations {
  const detail = useIssueDetail();
  const project = detail.relation.getProjectId(hostId) ?? detail.issue.getIssueById(hostId)?.project_id;
  const capture = useDetailScope(workspaceSlug, project, hostId);
  const refusal = useRefusalToast();
  return useMemo(
    () => ({
      copyLink: async (path: string) => copyUrlToClipboard(path),
      update: async (ws: string, projectId: string, id: string, body: IssueUpdate) => {
        const current = capture();
        if (!current()) return;
        await followInSession(() => detail.updateIssue(ws, projectId, id, body), {
          failed: (error) => {
            if (current() && !(error instanceof SessionChangedError)) refusal(error);
          },
        });
      },
      remove: (ws: string, projectId: string, id: string) => detail.removeIssue(ws, projectId, id),
    }),
    [capture, detail, refusal]
  );
}
