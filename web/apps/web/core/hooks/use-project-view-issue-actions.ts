/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { Issue, IssueCreate } from "@nerve/api-client";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import type { IssueChange, TLoader, TSupportedFilterForUpdate } from "@nerve/types";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueListOptions } from "@/lib/issue-filters";
import { settleEach, WRITES_PER_PROJECT } from "@/lib/settle-each";
import type { IssueDrop } from "@/store/issue/issue-writes";
import { useIssues } from "./store/use-issues";
import { useUserPermissions } from "./store/user";
import { useProjectViewPreferencesFollow } from "./use-project-view-preferences-follow";

/**
 * A project's view's page's list commands (P6a spec 2.5, R1; re-96 I3), every one a project list page has: the
 * workspace, the project and the view come from the page's route params, never the router store. The list is the route
 * project's under the page's draft of the view, read for any member of the project as the project page's is; the
 * writes are the project page's (a calendar's day added to as there, M4 I4), each into every list that shows the work
 * item, the view page's with them. A change of the page's display is the route view's draft's, whatever project its
 * caller names (the spreadsheet and the calendar name the project), followed as the project page's.
 */
export function useProjectViewIssueActions() {
  const { workspaceSlug, projectId: routeProjectId, viewId } = useParams();
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  const follow = useProjectViewPreferencesFollow();
  const permission = useUserPermissions();
  const role =
    workspaceSlug && routeProjectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, routeProjectId)
      : undefined;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !routeProjectId || !viewId || role === undefined) return;
      return issues.fetchIssues(workspaceSlug, routeProjectId, viewId, loadType, options);
    },
    [issues, workspaceSlug, routeProjectId, viewId, role]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !routeProjectId || role === undefined) return;
      return issues.fetchNextIssues(workspaceSlug, routeProjectId, groupId, subGroupId);
    },
    [issues, workspaceSlug, routeProjectId, role]
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
  const dropIssue = useCallback(
    async (targetProject: string, issueId: string, makeChange: (held: Issue) => IssueChange, drop: IssueDrop) => {
      if (!workspaceSlug) return;
      await issues.dropIssue(workspaceSlug, targetProject, issueId, makeChange, drop);
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
  // what the caller gives each of the work items of the route's project, and only that (a calendar's day, M4 I4)
  const addIssuesToView = useCallback(
    async (ids: readonly string[], extra: IssueChange = {}) => {
      if (!workspaceSlug || !routeProjectId) throw new SessionChangedError();
      return await settleEach(ids, WRITES_PER_PROJECT, (id) =>
        issues.updateIssue(workspaceSlug, routeProjectId, id, extra)
      );
    },
    [issues, workspaceSlug, routeProjectId]
  );
  const updateFilters = useCallback(
    async (_projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug || !viewId) return;
      return await follow(workspaceSlug, viewId, () =>
        issuesFilter.updateFilters(workspaceSlug, viewId, filterType, filters)
      );
    },
    [issuesFilter, workspaceSlug, viewId, follow]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      createIssueAsGiven: createIssue,
      updateIssue,
      removeIssue,
      addIssuesToView,
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
      addIssuesToView,
      archiveIssue,
      dropIssue,
      updateFilters,
    ]
  );
}
