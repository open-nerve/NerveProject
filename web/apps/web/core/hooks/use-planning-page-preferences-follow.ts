/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { useIssues } from "@/hooks/store/use-issues";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { PLANNING_PAGE_DOMAINS, type PlanningPageStoreType } from "./planning-page-domains";
import { usePagePreferencesFollow } from "./use-page-preferences-follow";

/**
 * Follows a confirmed change of a cycle's or a module's page's preferences as the project page's are (P5b spec 2.4;
 * P6b spec 2.2, 2.4): `id` is the cycle's or module's, and the page is the route's one of its project, held from the
 * change's sending to its answer (another one and back is not the same page). Its list is the route project's with
 * the page's term.
 */
export function usePlanningPagePreferencesFollow(storeType: PlanningPageStoreType) {
  const domain = PLANNING_PAGE_DOMAINS[storeType];
  const params = useParams();
  const { workspaceSlug, projectId } = params;
  const entityId = params[domain.param];
  const scope = useDetailScope(
    workspaceSlug ?? "",
    projectId,
    JSON.stringify([`${domain.name}-preferences`, entityId])
  );
  const { issues } = useIssues(storeType);
  const capture = useCallback(
    (slug: string, id: string) => {
      const owned = scope();
      return () => owned() && issues.rootIssueStore.active && workspaceSlug === slug && entityId === id;
    },
    [scope, issues, workspaceSlug, entityId]
  );
  const listed = useCallback(
    (id: string) => !projectId || issues.isPreferenceQueryCurrent({ projectID: projectId, entityID: id }),
    [issues, projectId]
  );
  const relist = useCallback(
    async (slug: string, id: string) => {
      if (projectId)
        await issues.fetchIssuesWithExistingPagination(slug, { projectID: projectId, entityID: id }, "mutation");
    },
    [issues, projectId]
  );
  return usePagePreferencesFollow({ capture, listed, relist });
}
