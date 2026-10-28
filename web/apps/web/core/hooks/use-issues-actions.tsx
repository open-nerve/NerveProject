/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { TLegacyIssueUpdateRequest } from "@nerve/types";
/* eslint-disable react-hooks/exhaustive-deps */
import { useCallback, useMemo } from "react";
// types
import { useParams } from "react-router";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EDraftIssuePaginationType } from "@nerve/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IssuePaginationOptions,
  TIssuesResponse,
  TLoader,
  TProfileViews,
  TSupportedFilterForUpdate,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "./store/use-issues";

export interface IssueActions {
  fetchIssues: (
    loadType: TLoader,
    options: IssuePaginationOptions,
    viewId?: string
  ) => Promise<TIssuesResponse | undefined>;
  fetchNextIssues: (groupId?: string, subGroupId?: string) => Promise<TIssuesResponse | undefined>;
  removeIssue: (projectId: string | undefined | null, issueId: string) => Promise<void>;
  createIssue?: (projectId: string | undefined | null, data: Partial<IssueDetail>) => Promise<Issue | undefined>;
  updateIssue?: (
    projectId: string | undefined | null,
    issueId: string,
    data: TLegacyIssueUpdateRequest
  ) => Promise<void>;
  removeIssueFromView?: (projectId: string | undefined | null, issueId: string) => Promise<void>;
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

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssuePaginationOptions) => {
      if (!workspaceSlug || !projectId) return;
      return issues.fetchIssues(workspaceSlug, projectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, projectId]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !projectId) return;
      return issues.fetchNextIssues(workspaceSlug, projectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, projectId]
  );

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: Partial<IssueDetail>) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, projectId, data);
    },
    [issues.createIssue, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: TLegacyIssueUpdateRequest) => {
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
      return await issuesFilter.updateFilters(workspaceSlug, projectId, filterType, filters);
    },
    [issuesFilter.updateFilters, workspaceSlug]
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

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssuePaginationOptions, viewId?: string) => {
      if (!workspaceSlug || !userId || !viewId) return;
      return issues.fetchIssues(workspaceSlug, userId, loadType, options, viewId as TProfileViews);
    },
    [issues.fetchIssues, workspaceSlug, userId]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !userId) return;
      return issues.fetchNextIssues(workspaceSlug, userId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, userId]
  );

  const createIssue = useCallback(
    async (projectId: string | undefined | null, data: Partial<IssueDetail>) => {
      if (!workspaceSlug || !projectId) return;
      return await issues.createIssue(workspaceSlug, projectId, data);
    },
    [issues.createIssue, workspaceSlug]
  );
  const updateIssue = useCallback(
    async (projectId: string | undefined | null, issueId: string, data: TLegacyIssueUpdateRequest) => {
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

  const fetchIssues = useCallback(
    async (loadType: TLoader, options: IssuePaginationOptions) => {
      if (!workspaceSlug || !projectId) return;
      return issues.fetchIssues(workspaceSlug, projectId, loadType, options);
    },
    [issues.fetchIssues, workspaceSlug, projectId]
  );
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (!workspaceSlug || !projectId) return;
      return issues.fetchNextIssues(workspaceSlug, projectId, groupId, subGroupId);
    },
    [issues.fetchIssues, workspaceSlug, projectId]
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
  const { workspaceSlug, globalViewId } = useParams();
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.WORKSPACE_DRAFT);
  const fetchIssues = useCallback(
    async (loadType: TLoader, _options: IssuePaginationOptions) => {
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

  const updateFilters = useCallback(
    async (projectId: string, filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) => {
      filters = filters as IIssueDisplayFilterOptions | IIssueDisplayProperties;
      if (!globalViewId || !workspaceSlug) return;
      return await issuesFilter.updateFilters(workspaceSlug, filterType, filters);
    },
    [globalViewId, workspaceSlug, issuesFilter]
  );

  return useMemo(
    () => ({
      fetchIssues,
      fetchNextIssues,
      createIssue,
      updateIssue,
      removeIssue,
      updateFilters,
    }),
    [fetchIssues, fetchNextIssues, createIssue, updateIssue, removeIssue, updateFilters]
  );
};
