/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { followInSession } from "@/lib/in-session";

/** Follow a confirmed preference change in the same session and project. List reads have their own refusal. */
export function useProjectPreferencesFollow() {
  const { workspaceSlug, projectId } = useParams();
  const capture = useDetailScope(workspaceSlug ?? "", projectId, "project-preferences");
  const { issues } = useIssues(EIssuesStoreType.PROJECT);
  const refusal = useRefusalToast();
  const toast = useRef(refusal);
  toast.current = refusal;
  return useCallback(
    async (slug: string, id: string, change: () => Promise<void>) => {
      const owned = capture();
      const current = () =>
        owned() &&
        issues.rootIssueStore.active &&
        issues.rootIssueStore.workspaceSlug === slug &&
        issues.rootIssueStore.projectId === id &&
        workspaceSlug === slug &&
        projectId === id;
      await followInSession(change, {
        failed: (error) => {
          if (current()) toast.current(error);
        },
        done: async () => {
          if (!current() || issues.isPreferenceQueryCurrent(id)) return;
          await followInSession(() => issues.fetchIssuesWithExistingPagination(slug, id, "mutation"), {
            // The binding retains read failures; the visible list owns its one refusal and GET-only retry.
            failed: () => undefined,
          });
        },
      });
    },
    [issues, capture, workspaceSlug, projectId]
  );
}
