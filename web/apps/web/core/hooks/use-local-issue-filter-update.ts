/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { EIssuesStoreType } from "@nerve/types";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useIssues } from "@/hooks/store/use-issues";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { followInSession } from "@/lib/in-session";
import type { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";

/** Local settings are saved synchronously; list reads retain their own visible error and GET-only retry. */
export function useLocalIssueFilterUpdate(
  workspaceSlug: string,
  projectId: string | undefined,
  targetId: string,
  member = false
) {
  const capture = useDetailScope(workspaceSlug, projectId, targetId, member);
  const { issues } = useIssues(member ? EIssuesStoreType.PROFILE : EIssuesStoreType.ARCHIVED);
  const refusal = useRefusalToast();
  return useCallback(
    async (change: () => Promise<unknown>): Promise<void> => {
      const current = capture();
      if (!current()) return;
      let reading: IssueListBinding | undefined = issues.currentList;
      await followInSession(
        async () => {
          const changed = change();
          reading = issues.currentList;
          return changed;
        },
        {
          failed: (error) => {
            if (!current() || error instanceof SessionChangedError || reading !== issues.currentList) return;
            // A binding-owned read refusal is already shown by the current list, not a failed local save.
            if (reading?.error !== error) refusal(error);
          },
        }
      );
    },
    [capture, issues, refusal]
  );
}
