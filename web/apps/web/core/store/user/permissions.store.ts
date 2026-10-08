/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { WorkspaceRole } from "@nerve/api-client";
import type { TUserPermissions, TUserPermissionsLevel } from "@nerve/constants";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { EUserProjectRoles } from "@nerve/types";
import { EUserWorkspaceRoles } from "@nerve/types";
// store
import type { RootStore } from "@/store/root.store";

type ETempUserRole = TUserPermissions | EUserWorkspaceRoles | EUserProjectRoles; // TODO: Remove this once user permissions are enums in @nerve/constants

export interface IUserPermissionStore {
  getWorkspaceRoleByWorkspaceSlug: (workspaceSlug: string) => WorkspaceRole | undefined;
  getProjectRoleByWorkspaceSlugAndProjectId: (
    workspaceSlug: string,
    projectId?: string
  ) => EUserPermissions | undefined;
  allowPermissions: (
    allowPermissions: ETempUserRole[],
    level: TUserPermissionsLevel,
    workspaceSlug?: string,
    projectId?: string,
    onPermissionAllowed?: () => boolean
  ) => boolean;
}

/**
 * @description The caller's permissions, as his pages check them (M3 design 7.3): his role in a workspace is the one
 * nerve gives with the workspace (Workspace.role), his role in a project the one nerve gives with the project
 * (Project.member_role, 7.2). Both are read from the stores of his workspaces and projects, which hold them by id and
 * no longer give a workspace he left, or its projects (v0 design 7.7).
 */
export class UserPermissionStore implements IUserPermissionStore {
  constructor(protected store: RootStore) {}

  /**
   * @description Returns the caller's role in the workspace, from the caller's workspaces; undefined while they are
   * not fetched, or for a workspace the caller is not a member of
   * @param { string } workspaceSlug
   * @returns { WorkspaceRole | undefined }
   */
  getWorkspaceRoleByWorkspaceSlug = computedFn(
    (workspaceSlug: string): WorkspaceRole | undefined =>
      this.store.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.role
  );

  /**
   * @description Returns the caller's role in a project of the workspace: none until he is its member (nerve gives
   * member_role null to one who only sees it), and then the admin's for a workspace admin, as nerve decides (3.4)
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { EUserPermissions | undefined }
   */
  getProjectRoleByWorkspaceSlugAndProjectId = computedFn(
    (workspaceSlug: string, projectId?: string): EUserPermissions | undefined => {
      const workspace = this.store.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
      const project = this.store.projectRoot.project.getProjectById(projectId);
      if (!workspace || project?.workspace_id !== workspace.id || project.member_role === null) return undefined;
      return workspace.role === EUserWorkspaceRoles.ADMIN ? EUserPermissions.ADMIN : project.member_role;
    }
  );

  // action helpers
  /**
   * @description Returns whether the user has the permission to perform an action
   * @param { TUserPermissions[] } allowPermissions
   * @param { TUserPermissionsLevel } level
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @param { () => boolean } onPermissionAllowed
   * @returns { boolean }
   */
  allowPermissions = (
    allowPermissions: ETempUserRole[],
    level: TUserPermissionsLevel,
    workspaceSlug?: string,
    projectId?: string,
    onPermissionAllowed?: () => boolean
  ): boolean => {
    const { workspaceSlug: currentWorkspaceSlug, projectId: currentProjectId } = this.store.router;
    if (!workspaceSlug) workspaceSlug = currentWorkspaceSlug;
    if (!projectId) projectId = currentProjectId;

    let currentUserRole: TUserPermissions | undefined = undefined;

    if (level === EUserPermissionsLevel.WORKSPACE) {
      currentUserRole = workspaceSlug ? this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
    }

    if (level === EUserPermissionsLevel.PROJECT) {
      currentUserRole = (workspaceSlug &&
        projectId &&
        this.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)) as EUserPermissions | undefined;
    }

    if (typeof currentUserRole === "string") {
      currentUserRole = parseInt(currentUserRole);
    }

    if (currentUserRole && typeof currentUserRole === "number" && allowPermissions.includes(currentUserRole)) {
      if (onPermissionAllowed) {
        return onPermissionAllowed();
      } else {
        return true;
      }
    }

    return false;
  };
}
