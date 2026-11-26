/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { usePagePreferencesFollow } from "./use-page-preferences-follow";

/**
 * Follows a change of a project's view's page's draft as the project page's preferences are followed (P6a spec 2.5):
 * `id` is the view's, and the page is the route's view of its project, held from the change to its end (another view
 * and back is not the same page); its list, the route project's under the draft, is read again when its query changed.
 */
export function useProjectViewPreferencesFollow() {
  const { workspaceSlug, projectId, viewId } = useParams();
  const scope = useDetailScope(workspaceSlug ?? "", projectId, JSON.stringify(["project-view-preferences", viewId]));
  const { issues } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  const capture = useCallback(
    (slug: string, id: string) => {
      const owned = scope();
      return () => owned() && issues.rootIssueStore.active && workspaceSlug === slug && viewId === id;
    },
    [scope, issues, workspaceSlug, viewId]
  );
  const listed = useCallback(
    (id: string) => !projectId || issues.isPreferenceQueryCurrent(projectId, id),
    [issues, projectId]
  );
  const relist = useCallback(
    async (slug: string, id: string) => {
      if (projectId) await issues.fetchIssuesWithExistingPagination(slug, projectId, id, "mutation");
    },
    [issues, projectId]
  );
  return usePagePreferencesFollow({ capture, listed, relist });
}
