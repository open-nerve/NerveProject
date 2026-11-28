/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { View } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { EUserWorkspaceRoles } from "@nerve/types";
// hooks
import { useUser, useUserPermissions } from "@/hooks/store/user";
// lib
import { workspaceViewsCreatableBy } from "@/lib/views-entry";
// store
import type { ViewContainer } from "@/store/views/view-container";

/** What the caller may do with the views of a container, as nerve decides it. */
type ViewRights = {
  /** Whether he is the container's admin (the project's, or the workspace's): he deletes any of its views. */
  isAdmin: boolean;
  /** Whether he owns the view and writes the container's views: he changes it and deletes it. */
  isOwner: (view: View) => boolean;
};

/**
 * The caller's rights over the views of the container (P6a spec 2.2; M7/P6b spec 2.7, 2.11, R15), by his role in it,
 * the permission store's by the route's workspace (and project), never the router store's: a project's admins delete
 * any of its views, and its owner, while the project's admin or member, changes it and deletes it (View.UpdatableBy,
 * View.DeletableBy, view.update, view.delete); likewise a workspace's own views by the role in the workspace
 * (workspace_view.update, workspace_view.delete, workspaceViewsCreatableBy), a project's role counting for none.
 */
export function useViewRights(workspaceSlug: string, container: ViewContainer): ViewRights {
  // store hooks
  const { getProjectRoleByWorkspaceSlugAndProjectId, getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { data: currentUser } = useUser();
  // the container's admin, and whether he writes its views
  let isAdmin: boolean, writes: boolean;
  if (container.kind === "project") {
    const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, container.projectId);
    isAdmin = role === EUserPermissions.ADMIN;
    writes = isAdmin || role === EUserPermissions.MEMBER;
  } else {
    const role = getWorkspaceRoleByWorkspaceSlug(workspaceSlug);
    isAdmin = role === EUserWorkspaceRoles.ADMIN;
    writes = workspaceViewsCreatableBy(role);
  }
  return { isAdmin, isOwner: (view) => writes && view.owned_by_id === currentUser?.id };
}
