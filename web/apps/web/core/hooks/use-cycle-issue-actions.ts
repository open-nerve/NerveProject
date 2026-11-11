/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import type { TLoader, TSupportedFilterForUpdate } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { CyclePage } from "@/store/issue/cycle/issue.store";
import type { IssueDrop } from "@/store/issue/root.store";
import { useIssues } from "./store/use-issues";
import { useUserPermissions } from "./store/user";
import { cycleListState } from "./use-cycle-list-state";
import { useCyclePreferencesFollow } from "./use-cycle-preferences-follow";
import { useCycleProgressFollow } from "./use-cycle-progress-follow";

/**
 * The cycle page's list commands (P5b spec 2.5): the workspace, project and cycle come from the page's route params,
 * never the router store. The list is read for any member of the project, as the project page's is (spec 2.1). Each
 * write it sends reads the cycle's progress again once nerve confirms it (useCycleProgressFollow, spec 2.6, R10).
 */
export function useCycleIssueActions() {
  const { workspaceSlug, projectId: routeProjectId, cycleId } = useParams();
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  const follow = useCyclePreferencesFollow();
  const followProgress = useCycleProgressFollow();
  const permission = useUserPermissions();
  const role =
    workspaceSlug && routeProjectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, routeProjectId)
      : undefined;
  const page = useMemo<CyclePage | undefined>(
    () => (routeProjectId && cycleId ? { projectID: routeProjectId, cycleID: cycleId } : undefined),
    [routeProjectId, cycleId]
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
   * A creation joins the page's cycle while the page shows the cycle taking work items as it is sent (spec 2.5): the
   * page's own entries offer it only then, and the app's own new work item, which resolves to these commands on a
   * cycle's page, otherwise creates as the project's page does, with no cycle the form could not drop when nerve
   * refused it. A creation of the cycle's project that joins it is the page's list's, whose confirmation reads the
   * cycle's progress again (spec 2.6, R10); one in another project, or while the cycle takes none, is not the cycle's.
   */
  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId || !page) return;
      const { planning, projectRoot } = issues.rootIssueStore.rootStore;
      const { joinable } = cycleListState(
        planning.cycles.getCycleById(page.cycleID),
        projectRoot.project.getProjectById(page.projectID)
      );
      if (!joinable || projectId !== page.projectID) return await issues.createIssue(workspaceSlug, projectId, data);
      return await followProgress(() => issues.createIssue(workspaceSlug, projectId, data, page));
    },
    [issues, workspaceSlug, page, followProgress]
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
   * The page's preferences are the cycle's, whatever project the caller names; their change is followed as the project
   * page's (P5b spec 2.4): its list read again when its query changed, a refusal said.
   */
  const updateFilters = useCallback(
    async (_projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug || !cycleId) return;
      return await follow(workspaceSlug, cycleId, () =>
        issuesFilter.updateFilters(workspaceSlug, cycleId, filterType, filters)
      );
    },
    [issuesFilter, workspaceSlug, cycleId, follow]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      archiveIssue,
      dropIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, archiveIssue, dropIssue, updateFilters]
  );
}
