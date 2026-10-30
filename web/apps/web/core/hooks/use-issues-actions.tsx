import type { IssueDrop } from "@/store/issue/root.store";
import type { PagedGroups } from "@/lib/paged-groups";
import type { IssueListOptions } from "@/lib/issue-filters";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";

/* eslint-disable react-hooks/exhaustive-deps */
import { useCallback, useMemo } from "react";
// types
import { useParams } from "react-router";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EUserPermissions } from "@nerve/constants";
import type { TLoader, TProfileViews, TSupportedFilterForUpdate } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "./store/use-issues";
import { useProjectPreferencesFollow } from "./use-project-preferences-follow";
import { useUserPermissions } from "./store/user";

export interface IssueActions {
  fetchIssues: (
    loadType: TLoader,
    options: IssueListOptions,
    viewId?: TProfileViews
  ) => Promise<PagedGroups | undefined>;
  fetchNextIssues: (groupId?: string, subGroupId?: string) => Promise<PagedGroups | undefined>;
  removeIssue: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  createIssue?: (projectId: string | undefined | null, data: IssueCreate) => Promise<Issue | undefined>;
  updateIssue?: (projectId: string | undefined | null, issueId: string, data: IssueUpdate) => Promise<void>;
  dropIssue?: (
    projectId: string,
    issueId: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop: IssueDrop
  ) => Promise<void>;
  archiveIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  restoreIssue?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  updateFilters: (
    projectId: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
}

export function useIssuesActions(storeType: EIssuesStoreType): IssueActions;
export function useIssuesActions(storeType: EIssuesStoreType | undefined): IssueActions | undefined;
export function useIssuesActions(storeType: EIssuesStoreType | undefined): IssueActions | undefined {
  const projectIssueActions = useProjectIssueActions();
  const profileIssueActions = useProfileIssueActions();
  const archivedIssueActions = useArchivedIssueActions();

  switch (storeType) {
    case undefined:
      return undefined;
    case EIssuesStoreType.PROFILE:
      return profileIssueActions;
    case EIssuesStoreType.ARCHIVED:
      return archivedIssueActions;
    case EIssuesStoreType.PROJECT:
    case EIssuesStoreType.DEFAULT:
      return projectIssueActions;
    default:
      throw new Error(`Unsupported issue store: ${storeType}`);
  }
}

const useProjectIssueActions = () => {
  // router
  const { workspaceSlug, projectId: routeProjectId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.PROJECT);
  const follow = useProjectPreferencesFollow();
  const permission = useUserPermissions();
  const role =
    workspaceSlug && routeProjectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, routeProjectId)
      : undefined;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !routeProjectId || role === undefined) return;
      return issues.fetchIssues(workspaceSlug, routeProjectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, routeProjectId, role]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !routeProjectId || role === undefined) return;
      return issues.fetchNextIssues(workspaceSlug, routeProjectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, routeProjectId, role]
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
      dropIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, archiveIssue, dropIssue, updateFilters]
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
    async (loadType: TLoader, options: IssueListOptions, viewId?: TProfileViews) => {
      if (!workspaceSlug || !userId || !viewId || !allowed) return;
      return issues.fetchIssues(workspaceSlug, userId, loadType, options, viewId);
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
      dropIssue,
      updateFilters,
    }),
    [fetchIssues, createIssue, updateIssue, removeIssue, archiveIssue, dropIssue, updateFilters]
  );
};

const useArchivedIssueActions = () => {
  // router
  const { workspaceSlug, projectId: routeProjectId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.ARCHIVED);
  const permission = useUserPermissions();
  const role =
    workspaceSlug && routeProjectId
      ? permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, routeProjectId)
      : undefined;
  const allowed = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssueListOptions) => {
      if (!workspaceSlug || !routeProjectId || !allowed) return;
      return issues.fetchIssues(workspaceSlug, routeProjectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, routeProjectId, allowed]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !routeProjectId || !allowed) return;
      return issues.fetchNextIssues(workspaceSlug, routeProjectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, routeProjectId, allowed]
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
