/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "@/hooks/store/use-issues";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { usePagePreferencesFollow } from "./use-page-preferences-follow";

/**
 * Follows a confirmed change of a cycle page's preferences as the project page's are (P5b spec 2.4): `id` is the
 * cycle's, and the page is the route's cycle of its project, held from the change's sending to its answer (another
 * cycle and back is not the same page). Its list is the route project's with the cycle.
 */
export function useCyclePreferencesFollow() {
  const { workspaceSlug, projectId, cycleId } = useParams();
  const scope = useDetailScope(workspaceSlug ?? "", projectId, JSON.stringify(["cycle-preferences", cycleId]));
  const { issues } = useIssues(EIssuesStoreType.CYCLE);
  const capture = useCallback(
    (slug: string, id: string) => {
      const owned = scope();
      return () => owned() && issues.rootIssueStore.active && workspaceSlug === slug && cycleId === id;
    },
    [scope, issues, workspaceSlug, cycleId]
  );
  const listed = useCallback(
    (id: string) => !projectId || issues.isPreferenceQueryCurrent({ projectID: projectId, cycleID: id }),
    [issues, projectId]
  );
  const relist = useCallback(
    async (slug: string, id: string) => {
      if (projectId)
        await issues.fetchIssuesWithExistingPagination(slug, { projectID: projectId, cycleID: id }, "mutation");
    },
    [issues, projectId]
  );
  return usePagePreferencesFollow({ capture, listed, relist });
}
