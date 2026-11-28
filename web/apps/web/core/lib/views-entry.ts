/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project, WorkspaceRole } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { EUserWorkspaceRoles } from "@nerve/types";

/**
 * Whether the caller is offered the ways into the project's views (M7 design 3.11; P6a spec 2.2, R14; M6-M7-07): the
 * sidebar's and the tab bar's views item, and the command palette's two commands that go to them. The project's views
 * are turned on (issue_views_view) and the caller has a role in it, an admin's, a member's or a guest's, as the
 * permission store gives it by the address's workspace and project (never the router's; the workspace's admin who is
 * its member is its admin): nerve lets all three read its views. As intakeEntryOn (intake-entry.ts), an archived
 * project is not left out: nerve reads its views as ever, and the shell shows its pages as archived. A store-free
 * module of lib, as intake-entry.ts is.
 */
export function viewEntryOn(project: Project | undefined, role: EUserPermissions | undefined): boolean {
  return (
    project !== undefined &&
    project.issue_views_view &&
    (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER || role === EUserPermissions.GUEST)
  );
}

/**
 * Whether the caller is offered a new view of the project (P6a spec 2.2, R4): the ways into its views are his
 * (viewEntryOn) and his role is an admin's or a member's, as nerve's view.create decides; a guest's is refused. The
 * command palette's create_view, and the views list's "Add view", its empty state's "New" and Save as read it.
 */
export function viewsCreatableBy(project: Project | undefined, role: EUserPermissions | undefined): boolean {
  return viewEntryOn(project, role) && (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER);
}

/**
 * Whether the caller is offered a workspace's own views (M7/P6b spec 2.7, R13; M7 design 3.7): those of no project,
 * listed and read, on the views list page, the view page's switcher and a saved view's page. His role in the workspace
 * is an admin's or a member's, as nerve's workspace_view.read decides; a guest's is refused (forbidden), and one with
 * no role there (not its member, or its workspaces not yet fetched) has none. A workspace's views have no switch. The
 * role is the permission store's by the route's workspace (getWorkspaceRoleByWorkspaceSlug), or the workspace store's
 * workspace's (Workspace.role): the same.
 */
export function workspaceViewsOn(role: WorkspaceRole | undefined): boolean {
  return role === EUserWorkspaceRoles.ADMIN || role === EUserWorkspaceRoles.MEMBER;
}

/**
 * Whether the caller writes a workspace's own views (M7/P6b spec 2.7): makes a new one ("Add view", Save as), and
 * changes and deletes his own (Update view, the edit and the deletion of the views he owns). His role in the workspace
 * is an admin's or a member's, as nerve's workspace_view.create, workspace_view.update and workspace_view.delete
 * decide: as workspaceViewsOn today, kept apart so that a change of who writes is made here alone.
 */
export function workspaceViewsCreatableBy(role: WorkspaceRole | undefined): boolean {
  return role === EUserWorkspaceRoles.ADMIN || role === EUserWorkspaceRoles.MEMBER;
}
