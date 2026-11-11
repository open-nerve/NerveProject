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
import { useCyclePreferencesFollow } from "./use-cycle-preferences-follow";

/**
 * The cycle page's list commands (P5b spec 2.5): the workspace, project and cycle come from the page's route params,
 * never the router store. The list is read for any member of the project, as the project page's is (spec 2.1).
 */
export function useCycleIssueActions() {
  const { workspaceSlug, projectId: routeProjectId, cycleId } = useParams();
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  const follow = useCyclePreferencesFollow();
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
  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId || !page) return;
      return await issues.createIssue(workspaceSlug, projectId, data, page);
    },
    [issues, workspaceSlug, page]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.updateIssue(workspaceSlug, projectId, issueId, data);
    },
    [issues, workspaceSlug]
  );
  const dropIssue = useCallback(
    async (targetProject: string, issueId: string, makeBody: (held: Issue) => IssueUpdate, drop: IssueDrop) => {
      if (!workspaceSlug) return;
      await issues.dropIssue(workspaceSlug, targetProject, issueId, makeBody, drop);
    },
    [issues, workspaceSlug]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.removeIssue(workspaceSlug, projectId, issueId);
    },
    [issues, workspaceSlug]
  );
  const archiveIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.archiveIssue(workspaceSlug, projectId, issueId);
    },
    [issues, workspaceSlug]
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
