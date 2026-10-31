import type { TProfileViews } from "@nerve/types";
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useParams } from "react-router";
import { EUserPermissions } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import { toMemberFilters, type IssueListOptions } from "@/lib/issue-filters";
import type { IssueRootStore } from "@/store/issue/root.store";
import type { IssueActions } from "@/hooks/use-issues-actions";
import { useUserPermissions } from "@/hooks/store/user";
import { useSessionSWR } from "@/lib/use-session-swr";

/** A mounted layout's first page belongs to its session, scope, view and query. */
export function useIssueListMount(
  type: EIssuesStoreType,
  root: IssueRootStore,
  fetchIssues: IssueActions["fetchIssues"],
  options: IssueListOptions | undefined,
  viewId?: TProfileViews,
  variant = ""
) {
  const { workspaceSlug, projectId, userId } = useParams();
  const permission = useUserPermissions();
  const profile = type === EIssuesStoreType.PROFILE;
  const target = profile ? userId : projectId;
  const role = workspaceSlug
    ? profile
      ? permission.getWorkspaceRoleByWorkspaceSlug(workspaceSlug)
      : permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
    : undefined;
  const allowed =
    profile || type === EIssuesStoreType.ARCHIVED
      ? role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER
      : role !== undefined;
  const workspaceID = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id : undefined;
  const projectIDs = profile
    ? (toMemberFilters(
        userId ? (root.profileIssuesFilter.getIssueFilters(userId, workspaceSlug)?.richFilters ?? {}) : {}
      ).project_id ?? [])
    : projectId
      ? [projectId]
      : [];
  const identity = root.permissionReads.identity(workspaceID, projectIDs, profile);
  const key =
    options && allowed && (!profile || viewId)
      ? (["ISSUE_LIST", type, workspaceSlug, target, viewId ?? "", variant, identity, JSON.stringify(options)] as const)
      : null;
  return useSessionSWR(key, () => fetchIssues("init-loader", options!, viewId), { dedupingInterval: 0 });
}
