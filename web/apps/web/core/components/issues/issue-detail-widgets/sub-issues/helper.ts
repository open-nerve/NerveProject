/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import type { TSubIssueOperations } from "@nerve/types";
import { copyUrlToClipboard } from "@nerve/utils";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "../use-detail-scope";

export function useSubIssueOperations(workspaceSlug: string, projectId: string, hostId: string): TSubIssueOperations {
  const detail = useIssueDetail();
  const capture = useDetailScope(workspaceSlug, projectId, hostId);
  const refusal = useRefusalToast();
  return useMemo(() => {
    const run = async (operation: () => Promise<void>, parent: string, child: string) => {
      const current = capture();
      const store = detail.subIssues;
      if (!current()) return;
      store.setSubIssueHelpers(parent, "issue_loader", child);
      await followInSession(operation, {
        failed: (error) => {
          if (current() && !(error instanceof SessionChangedError)) refusal(error);
        },
      });
      if (current() && store.subIssueHelpersByIssueId(parent).issue_loader.includes(child))
        store.setSubIssueHelpers(parent, "issue_loader", child);
    };
    return {
      copyLink: async (path) => copyUrlToClipboard(path),
      // These callers own the dialog/read error; preserve rejection so failure cannot close the form.
      fetchSubIssues: async (ws, project, parent) => {
        await detail.fetchSubIssues(ws, project, parent);
      },
      addSubIssue: (ws, project, parent, ids) => detail.createSubIssues(ws, project, parent, ids),
      updateSubIssue: (ws, project, parent, id, body) =>
        run(() => detail.updateSubIssue(ws, project, parent, id, body), parent, id),
      removeSubIssue: (ws, project, parent, id) =>
        run(() => detail.removeSubIssue(ws, project, parent, id), parent, id),
      deleteSubIssue: (ws, project, parent, id) => detail.deleteSubIssue(ws, project, parent, id),
    };
  }, [capture, detail, refusal]);
}
