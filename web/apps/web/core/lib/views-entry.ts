/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";

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
