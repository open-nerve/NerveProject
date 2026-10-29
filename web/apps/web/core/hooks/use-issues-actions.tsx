import type { PagedGroups } from "@/lib/paged-groups";
import type { IssueListOptions } from "@/lib/issue-filters";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";

import type { IssueDetail } from "@nerve/api-client";
/* eslint-disable react-hooks/exhaustive-deps */
import { useCallback, useMemo } from "react";
// types
import { useParams } from "react-router";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EDraftIssuePaginationType, EUserPermissions } from "@nerve/constants";
import type { TLoader, TProfileViews, TSupportedFilterForUpdate } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "./store/use-issues";
import { useProjectPreferencesFollow } from "./use-project-preferences-follow";
import { useUserPermissions } from "./store/user";

export interface IssueActions {
  fetchIssues: (loadType: TLoader, options: IssueListOptions, viewId?: string) => Promise<PagedGroups | undefined>;
  fetchNextIssues: (groupId?: string, subGroupId?: string) => Promise<PagedGroups | undefined>;
  removeIssue: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  createIssue?: (projectId: string | undefined | null, data: IssueCreate) => Promise<Issue | undefined>;
  updateIssue?: (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => Promise<void>;
  archiveIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  restoreIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  updateFilters: (
    projectId: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
}

export const useIssuesActions = (storeType: EIssuesStoreType): IssueActions => {
  const projectIssueActions = useProjectIssueActions();
  const profileIssueActions = useProfileIssueActions();
  const archivedIssueActions = useArchivedIssueActions();
  const workspaceDraftIssueActions = useWorkspaceDraftIssueActions();

  switch (storeType) {
    case EIssuesStoreType.PROFILE:
      return profileIssueActions;
    case EIssuesStoreType.ARCHIVED:
      return archivedIssueActions;
    case EIssuesStoreType.WORKSPACE_DRAFT:
      //@ts-expect-error type mismatch
      return workspaceDraftIssueActions;
    case EIssuesStoreType.PROJECT:
    case EIssuesStoreType.DEFAULT:
      return projectIssueActions;
    default:
      throw new Error(`Unsupported issue store: ${storeType}`);
  }
};

const useProjectIssueActions = () => {
  // router
  const { workspaceSlug, projectId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROJECT);
  const follow = useProjectPreferencesFollow();
  const permission = useUserPermissions();
  const role =
    workspaceSlug && projectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !projectId || role === undefined) return;
      return issues.fetchIssues(workspaceSlug, projectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, projectId, role]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !projectId || role === undefined) return;
      return issues.fetchNextIssues(workspaceSlug, projectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, projectId, role]
  );

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, projectId, data);
    },
    [issues.createIssue, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.updateIssue(workspaceSlug, projectId, issueId, data);
    },
    [issues.updateIssue, workspaceSlug]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.removeIssue(workspaceSlug, projectId, issueId);
    },
    [issues.removeIssue, workspaceSlug]
  );
  const archiveIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.archiveIssue(workspaceSlug, projectId, issueId);
    },
    [issues.archiveIssue, workspaceSlug]
  );

  const updateFilters = useCallback(
    async (projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug) return;
      return await follow(workspaceSlug, projectId, () =>
        issuesFilter.updateFilters(workspaceSlug, projectId, filterType, filters)
      );
    },
    [issuesFilter.updateFilters, workspaceSlug, follow]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      archiveIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, archiveIssue, updateFilters]
  );
};

const useProfileIssueActions = () => {
  // router
  const { workspaceSlug, userId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROFILE);
  const permission = useUserPermissions();
  const role = workspaceSlug ? permission.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
  const allowed = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions, viewId?: string) => {
      if (!workspaceSlug || !userId || !viewId || !allowed) return;
      return issues.fetchIssues(workspaceSlug, userId, loadType, options, viewId as TProfileViews);
    },
    [issues.fetchIssues, workspaceSlug, userId, allowed]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string) => {
      if (!workspaceSlug || !userId || !allowed) return;
      return issues.fetchNextIssues(workspaceSlug, userId, groupId);
    },
    [issues.fetchIssues, workspaceSlug, userId, allowed]
  );

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: IssueCreate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, projectId, data);
    },
    [issues.createIssue, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.updateIssue(workspaceSlug, projectId, issueId, data);
    },
    [issues.updateIssue, workspaceSlug]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.removeIssue(workspaceSlug, projectId, issueId);
    },
    [issues.removeIssue, workspaceSlug]
  );
  const archiveIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.archiveIssue(workspaceSlug, projectId, issueId);
    },
    [issues.archiveIssue, workspaceSlug]
  );

  const updateFilters = useCallback(
    async (projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!userId || !workspaceSlug) return;
      return await issuesFilter.updateFilters(workspaceSlug, projectId, filterType, filters, userId);
    },
    [issuesFilter.updateFilters, userId, workspaceSlug]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      archiveIssue,
      updateFilters,
    }),
    [fetchIssues, createIssue, updateIssue, removeIssue, archiveIssue, updateFilters]
  );
};

const useArchivedIssueActions = () => {
  // router
  const { workspaceSlug, projectId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.ARCHIVED);
  const permission = useUserPermissions();
  const role =
    workspaceSlug && projectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;
  const allowed = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !projectId || !allowed) return;
      return issues.fetchIssues(workspaceSlug, projectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, projectId, allowed]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !projectId || !allowed) return;
      return issues.fetchNextIssues(workspaceSlug, projectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, projectId, allowed]
  );

  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.removeIssue(workspaceSlug, projectId, issueId);
    },
    [issues.removeIssue]
  );
  const restoreIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.restoreIssue(workspaceSlug, projectId, issueId);
    },
    [issues.restoreIssue]
  );

  const updateFilters = useCallback(
    async (projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      if (!workspaceSlug) return;
      return await issuesFilter.updateFilters(workspaceSlug, projectId, filterType, filters);
    },
    [issuesFilter.updateFilters]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      removeIssue,
      restoreIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, removeIssue, restoreIssue, updateFilters]
  );
};

const useWorkspaceDraftIssueActions = () => {
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { issues } = useIssues(EIssuesStoreType.WORKSPACE_DRAFT);
  const fetchIssues = useCallback(
    async (loadType: TLoader, _options: IssueListOptions) => {
      if (!workspaceSlug) return;
      return issues.fetchIssues(workspaceSlug, loadType, EDraftIssuePaginationType.INIT);
    },
    [workspaceSlug, issues]
  );

  const fetchNextIssues = useCallback(async () => {
    if (!workspaceSlug) return;
    return issues.fetchIssues(workspaceSlug, "pagination", EDraftIssuePaginationType.NEXT);
  }, [workspaceSlug, issues]);

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: Partial<IssueDetail>) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, data);
    },
    [issues, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: Partial<IssueDetail>) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.updateIssue(workspaceSlug, issueId, data);
    },
    [issues, workspaceSlug]
  );
  const removeIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.removeIssue(issueId);
    },
    [issues, workspaceSlug]
  );

  // const moveToIssue = useCallback(
  //   async (workspaceSlug: string, issueId: string, data: Partial<IssueDetail>) => {
  //     if (!workspaceSlug || !issueId || !data) return;
  //     return await issues.moveToIssues(workspaceSlug, issueId, data);
  //   },
  //   [issues]
  // );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue]
  );
};
