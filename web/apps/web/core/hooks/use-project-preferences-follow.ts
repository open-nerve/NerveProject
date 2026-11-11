/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { usePagePreferencesFollow } from "./use-page-preferences-follow";

/** Follow a confirmed preference change in the same session and project. List reads have their own refusal. */
export function useProjectPreferencesFollow() {
  const { workspaceSlug, projectId } = useParams();
  const scope = useDetailScope(workspaceSlug ?? "", projectId, "project-preferences");
  const { issues } = useIssues(EIssuesStoreType.PROJECT);
  const capture = useCallback(
    (slug: string, id: string) => {
      const owned = scope();
      return () =>
        owned() &&
        issues.rootIssueStore.active &&
        issues.rootIssueStore.workspaceSlug === slug &&
        issues.rootIssueStore.projectId === id &&
        workspaceSlug === slug &&
        projectId === id;
    },
    [scope, issues, workspaceSlug, projectId]
  );
  const listed = useCallback((id: string) => issues.isPreferenceQueryCurrent(id), [issues]);
  const relist = useCallback(
    (slug: string, id: string) => issues.fetchIssuesWithExistingPagination(slug, id, "mutation"),
    [issues]
  );
  return usePagePreferencesFollow({ capture, listed, relist });
}
