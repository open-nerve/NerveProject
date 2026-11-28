/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import { usePagePreferencesFollow } from "./use-page-preferences-follow";

/**
 * Follows a change of a workspace's view's page's preferences as a project's view's page's are followed (M7/P6b spec
 * 2.3): `id` is the page's (page-id.ts), and the page is the route's view of its workspace, held from the change to its
 * end; its list, the workspace's under the page's preferences, is read again when its query changed.
 */
export function useGlobalPreferencesFollow() {
  const { workspaceSlug, globalViewId } = useParams();
  const scope = useDetailScope(
    workspaceSlug ?? "",
    undefined,
    JSON.stringify(["workspace-view-preferences", globalViewId])
  );
  const { issues } = useIssues(EIssuesStoreType.GLOBAL);
  const capture = useCallback(
    (slug: string, id: string) => {
      const owned = scope();
      return () =>
        owned() &&
        issues.rootIssueStore.active &&
        workspaceSlug === slug &&
        !!globalViewId &&
        workspacePageId(slug, globalViewId) === id;
    },
    [scope, issues, workspaceSlug, globalViewId]
  );
  const listed = useCallback(
    () => !workspaceSlug || !globalViewId || issues.isPreferenceQueryCurrent(workspaceSlug, globalViewId),
    [issues, workspaceSlug, globalViewId]
  );
  const relist = useCallback(
    async (slug: string) => {
      if (globalViewId) await issues.fetchIssuesWithExistingPagination(slug, globalViewId);
    },
    [issues, globalViewId]
  );
  return usePagePreferencesFollow({ capture, listed, relist });
}
