/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext } from "react";
import { useParams } from "react-router";
import type { IIssueFilters } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import { workspacePageId } from "@/store/issue/workspace/page-id";

/** The store types of the pages whose layouts list work items under the page's own filters. */
type PageStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.PROFILE
  | EIssuesStoreType.ARCHIVED
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.GLOBAL;

/**
 * The filters of the list page in the route, by its store type (P5b spec 2.5): the project's page, a profile and the
 * archive read them as they always have; a cycle's page reads its own preferences by the route's `cycleId`, and a
 * module's page its own by the route's `moduleId` (P6b spec 2.6), and a project's view's page its draft of the view by the
 * route's `viewId` (P6a spec 2.5), and a workspace's view's page its own by the page id of the route's `globalViewId`
 * (M7/P6b spec 2.4).
 */
export function usePageIssueFilters(storeType: PageStoreType): IIssueFilters | undefined {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("usePageIssueFilters must be used within StoreProvider");
  const { workspaceSlug, userId, cycleId, moduleId, viewId, globalViewId } = useParams();
  const root = context.issue;
  switch (storeType) {
    case EIssuesStoreType.PROJECT:
      return root.projectIssuesFilter.issueFilters;
    case EIssuesStoreType.PROFILE:
      return root.profileIssuesFilter.getIssueFilters(userId ?? "", workspaceSlug);
    case EIssuesStoreType.ARCHIVED:
      return root.archivedIssuesFilter.issueFilters;
    case EIssuesStoreType.CYCLE:
      return cycleId ? root.cycleIssuesFilter.getIssueFilters(cycleId) : undefined;
    case EIssuesStoreType.MODULE:
      return moduleId ? root.moduleIssuesFilter.getIssueFilters(moduleId) : undefined;
    case EIssuesStoreType.PROJECT_VIEW:
      return viewId ? root.projectViewIssuesFilter.getIssueFilters(viewId) : undefined;
    case EIssuesStoreType.GLOBAL:
      return workspaceSlug && globalViewId
        ? root.globalIssuesFilter.getIssueFilters(workspacePageId(workspaceSlug, globalViewId))
        : undefined;
  }
}
