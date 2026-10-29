/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";

/** Follow a confirmed preference change in the same session and project. List reads have their own refusal. */
export function useProjectPreferencesFollow() {
  const { workspaceSlug, projectId } = useParams();
  const scope = useRef({ workspaceSlug, projectId });
  scope.current = { workspaceSlug, projectId };
  const { issues } = useIssues(EIssuesStoreType.PROJECT);
  const refusal = useRefusalToast();
  const toast = useRef(refusal);
  toast.current = refusal;
  return useCallback(
    async (slug: string, id: string, change: () => Promise<void>) => {
      const current = () =>
        issues.rootIssueStore.active &&
        issues.rootIssueStore.workspaceSlug === slug &&
        issues.rootIssueStore.projectId === id &&
        scope.current.workspaceSlug === slug &&
        scope.current.projectId === id;
      await followInSession(change, {
        failed: (error) => {
          if (current()) toast.current(error);
        },
        done: async () => {
          if (!current() || issues.isPreferenceQueryCurrent(id)) return;
          await followInSession(() => issues.fetchIssuesWithExistingPagination(slug, id, "mutation"), {
            failed: (error) => {
              if (current()) toast.current(error);
            },
          });
        },
      });
    },
    [issues]
  );
}
