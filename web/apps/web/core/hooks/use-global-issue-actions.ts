/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { IssueCreate } from "@nerve/api-client";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import type { IssueChange, TLoader, TSupportedFilterForUpdate } from "@nerve/types";
import type { IssueListOptions } from "@/lib/issue-filters";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import { useIssues } from "./store/use-issues";
import { useUserPermissions } from "./store/user";
import { useGlobalPreferencesFollow } from "./use-global-preferences-follow";

/**
 * A workspace's view's page's list commands (M7/P6b spec 2.3, R1): the workspace and the view come from the page's
 * route params, never the router store. The list is the workspace's under the page's preferences, read for any role
 * of the workspace; the writes are in the project each names, as a person's page's, each into every list that shows
 * the work item, the workspace's with them. A change of the page's display is the route page's, whatever id its caller
 * names (the spreadsheet names the route's project, none here), followed by a read of the list. No calendar's day to
 * add to (addIssuesToView), no drag (dropIssue): the page draws the spreadsheet alone.
 */
export function useGlobalIssueActions() {
  const { workspaceSlug, globalViewId } = useParams();
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.GLOBAL);
  const follow = useGlobalPreferencesFollow();
  const permission = useUserPermissions();
  const role = workspaceSlug ? permission.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !globalViewId || role === undefined) return;
      return issues.fetchIssues(workspaceSlug, globalViewId, loadType, options);
    },
    [issues, workspaceSlug, globalViewId, role]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string) => {
      if (!workspaceSlug || role === undefined) return;
      return issues.fetchNextIssues(workspaceSlug, groupId);
    },
    [issues, workspaceSlug, role]
  );
  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, projectId, data);
    },
    [issues, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, change: IssueChange) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.updateIssue(workspaceSlug, projectId, issueId, change);
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
  const updateFilters = useCallback(
    async (_projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug || !globalViewId) return;
      const page = workspacePageId(workspaceSlug, globalViewId);
      return await follow(workspaceSlug, page, () =>
        issuesFilter.updateFilters(workspaceSlug, page, filterType, filters)
      );
    },
    [issuesFilter, workspaceSlug, globalViewId, follow]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      createIssueAsGiven: createIssue,
      updateIssue,
      removeIssue,
      archiveIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, archiveIssue, updateFilters]
  );
}
