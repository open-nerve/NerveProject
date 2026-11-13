/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import type { TLoader, TSupportedFilterForUpdate } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { PlanningPage } from "@/store/issue/helpers/planning-page-issues.store";
import type { IssueDrop } from "@/store/issue/issue-writes";
import { useIssues } from "./store/use-issues";
import { useUserPermissions } from "./store/user";
import { PLANNING_PAGE_DOMAINS, type PlanningPageStoreType } from "./planning-page-domains";
import { usePlanningPagePreferencesFollow } from "./use-planning-page-preferences-follow";
import { usePlanningPageProgressFollow } from "./use-planning-page-progress-follow";

/**
 * A cycle's or a module's page's list commands (P5b spec 2.5; P6b spec 2.2): the workspace, project and cycle or
 * module come from the page's route params, never the router store. The list is read for any member of the project, as
 * the project page's is (P5b spec 2.1). Each write it sends reads the page's progress again once nerve confirms it
 * (usePlanningPageProgressFollow, P5b spec 2.6, R10).
 */
export function usePlanningPageIssueActions(storeType: PlanningPageStoreType) {
  const domain = PLANNING_PAGE_DOMAINS[storeType];
  const params = useParams();
  const { workspaceSlug, projectId: routeProjectId } = params;
  const entityId = params[domain.param];
  const { issues, issuesFilter } = useIssues(storeType);
  const follow = usePlanningPagePreferencesFollow(storeType);
  const followProgress = usePlanningPageProgressFollow(storeType);
  const permission = useUserPermissions();
  const role =
    workspaceSlug && routeProjectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, routeProjectId)
      : undefined;
  const page = useMemo<PlanningPage | undefined>(
    () => (routeProjectId && entityId ? { projectID: routeProjectId, entityID: entityId } : undefined),
    [routeProjectId, entityId]
  );

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !page || role === undefined) return;
      return issues.fetchIssues(workspaceSlug, page, loadType, options);
    },
    [issues, workspaceSlug, page, role]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !page || role === undefined) return;
      return issues.fetchNextIssues(workspaceSlug, page, groupId, subGroupId);
    },
    [issues, workspaceSlug, page, role]
  );
  /**
   * A creation joins the page's cycle or module while the page shows it taking work items as it is sent (P5b spec
   * 2.5): the page's own entries offer it only then, and the app's own new work item, which resolves to these commands
   * on the page, otherwise creates as the project's page does, with nothing of the page the form could not drop when
   * nerve refused it. A creation of the page's project that joins it is the page's list's, whose confirmation reads
   * the progress again (P5b spec 2.6, R10); one in another project, or while the page takes none, is not the page's.
   */
  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId || !page) return;
      if (!domain.joinable(issues.rootIssueStore.rootStore, page) || projectId !== page.projectID)
        return await issues.createIssue(workspaceSlug, projectId, data);
      return await followProgress(() => issues.createIssue(workspaceSlug, projectId, data, page));
    },
    [issues, domain, workspaceSlug, page, followProgress]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => {
      if (!workspaceSlug || !projectId) return;
      return await followProgress(() => issues.updateIssue(workspaceSlug, projectId, issueId, data));
    },
    [issues, workspaceSlug, followProgress]
  );
  const dropIssue = useCallback(
    async (targetProject: string, issueId: string, makeBody: (held: Issue) => IssueUpdate, drop: IssueDrop) => {
      if (!workspaceSlug) return;
      await followProgress(() => issues.dropIssue(workspaceSlug, targetProject, issueId, makeBody, drop));
    },
    [issues, workspaceSlug, followProgress]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await followProgress(() => issues.removeIssue(workspaceSlug, projectId, issueId));
    },
    [issues, workspaceSlug, followProgress]
  );
  const archiveIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await followProgress(() => issues.archiveIssue(workspaceSlug, projectId, issueId));
    },
    [issues, workspaceSlug, followProgress]
  );
  /**
   * Takes a work item of the page's project out of the page's cycle or module, as the domain sends it (P6b spec 2.2,
   * 2.5): a write of the page's list, whose confirmation reads the progress again. A work item of another project is
   * in none of its cycles or modules.
   */
  const removeIssueFromView = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !page || projectId !== page.projectID) return;
      await followProgress(() => domain.remove(issues.rootIssueStore.rootStore, workspaceSlug, page, issueId));
    },
    [issues, domain, workspaceSlug, page, followProgress]
  );
  /**
   * The page's preferences are its cycle's or module's, whatever project the caller names; their change is followed
   * as the project page's (P5b spec 2.4): its list read again when its query changed, a refusal said.
   */
  const updateFilters = useCallback(
    async (_projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug || !entityId) return;
      return await follow(workspaceSlug, entityId, () =>
        issuesFilter.updateFilters(workspaceSlug, entityId, filterType, filters)
      );
    },
    [issuesFilter, workspaceSlug, entityId, follow]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      removeIssueFromView,
      archiveIssue,
      dropIssue,
      updateFilters,
    }),
    [
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      removeIssueFromView,
      archiveIssue,
      dropIssue,
      updateFilters,
    ]
  );
}
