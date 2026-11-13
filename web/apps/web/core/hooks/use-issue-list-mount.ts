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
import { isPlanningPageStoreType, PLANNING_PAGE_DOMAINS } from "./planning-page-domains";

/** A mounted layout's first page belongs to its session, scope, page, view and query. */
export function useIssueListMount(
  type: EIssuesStoreType,
  root: IssueRootStore,
  fetchIssues: IssueActions["fetchIssues"],
  options: IssueListOptions | undefined,
  viewId?: TProfileViews,
  variant = ""
) {
  const params = useParams();
  const { workspaceSlug, projectId, userId } = params;
  const permission = useUserPermissions();
  const profile = type === EIssuesStoreType.PROFILE;
  const target = profile ? userId : projectId;
  // A cycle's or a module's page lists its project with the cycle or module (P5b spec 2.1; P6b spec 2.3): each cycle
  // and each module is its own list, none until the route names it.
  const page = isPlanningPageStoreType(type) ? params[PLANNING_PAGE_DOMAINS[type].param] : "";
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
      ? ([
          "ISSUE_LIST",
          type,
          workspaceSlug,
          target,
          page,
          viewId ?? "",
          variant,
          identity,
          JSON.stringify(options),
        ] as const)
      : null;
  return useSessionSWR(key, () => fetchIssues("init-loader", options!, viewId), { dedupingInterval: 0 });
}
