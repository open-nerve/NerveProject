/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { unset, set } from "lodash-es";
import { action, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, WorkspaceRole } from "@nerve/api-client";
import type { TUserPermissions, TUserPermissionsLevel } from "@nerve/constants";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { EUserProjectRoles, IUserProjectsRole, TProjectMembership } from "@nerve/types";
import { EUserWorkspaceRoles } from "@nerve/types";
// services
import { WorkspaceService } from "@/services/workspace.service";
import type { RootStore } from "@/store/root.store";
import projectMemberService from "@/services/project/project-member.service";
import { UserService } from "@/services/user.service";

type ETempUserRole = TUserPermissions | EUserWorkspaceRoles | EUserProjectRoles; // TODO: Remove this once user permissions are enums in @nerve/constants

export interface IUserPermissionStore {
  // observables
  projectUserInfo: Record<string, Record<string, TProjectMembership>>; // workspaceSlug -> projectId -> TProjectMembership
  workspaceProjectsPermissions: Record<string, IUserProjectsRole>; // workspaceSlug -> IUserProjectsRole
  // computed helpers
  getWorkspaceRoleByWorkspaceSlug: (workspaceSlug: string) => WorkspaceRole | undefined;
  getProjectRolesByWorkspaceSlug: (workspaceSlug: string) => IUserProjectsRole;
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
  // actions
  fetchUserProjectInfo: (workspaceSlug: string, projectId: string) => Promise<TProjectMembership>;
  fetchUserProjectPermissions: (workspaceSlug: string) => Promise<IUserProjectsRole>;
  joinProject: (workspaceSlug: string, projectId: string) => Promise<void>;
  leaveProject: (workspaceSlug: string, projectId: string) => Promise<void>;
}

/**
 * @description This store is used to handle permission layer for the currently logged user.
 * It manages workspace and project level permissions, roles and access control. The caller's role in a workspace
 * is the one nerve gives with the workspace (Workspace.role, M3 design 7.2).
 */
export class UserPermissionStore implements IUserPermissionStore {
  // constants
  projectUserInfo: Record<string, Record<string, TProjectMembership>> = {};
  workspaceProjectsPermissions: Record<string, IUserProjectsRole> = {};
  // services
  userService: UserService;
  private readonly workspaceService = new WorkspaceService();
  // observables

  constructor(
    protected store: RootStore,
    api: ApiClient
  ) {
    makeObservable(this, {
      // observables
      projectUserInfo: observable,
      workspaceProjectsPermissions: observable,
      // computed
      // actions
      fetchUserProjectInfo: action,
      fetchUserProjectPermissions: action,
      joinProject: action,
      leaveProject: action,
    });
    // services
    this.userService = new UserService(api);
  }

  // computed helpers
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
   * @description Returns the project membership permission
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { EUserPermissions | undefined }
   */
  protected getProjectRole = computedFn((workspaceSlug: string, projectId?: string): EUserPermissions | undefined => {
    if (!workspaceSlug || !projectId) return undefined;
    const projectRole = this.workspaceProjectsPermissions?.[workspaceSlug]?.[projectId];
    if (!projectRole) return undefined;
    const workspaceRole = this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug);
    if (workspaceRole === EUserWorkspaceRoles.ADMIN) return EUserPermissions.ADMIN;
    else return projectRole;
  });

  /**
   * @description Returns the project permissions by workspace slug
   * @param { string } workspaceSlug
   * @returns { IUserProjectsRole }
   */
  getProjectRolesByWorkspaceSlug = computedFn((workspaceSlug: string): IUserProjectsRole => {
    const projectPermissions = this.workspaceProjectsPermissions[workspaceSlug] || {};
    return Object.keys(projectPermissions).reduce((acc, projectId) => {
      const projectRole = this.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
      if (projectRole) {
        acc[projectId] = projectRole;
      }
      return acc;
    }, {} as IUserProjectsRole);
  });

  /**
   * @description Returns the current project permissions
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { EUserPermissions | undefined }
   */
  getProjectRoleByWorkspaceSlugAndProjectId = computedFn(
    (workspaceSlug: string, projectId?: string): EUserPermissions | undefined =>
      this.getProjectRole(workspaceSlug, projectId)
  );

  /**
   * @description Fetches project-level entities that are not automatically loaded by the project wrapper.
   * This is used when joining a project to ensure all necessary workspace-level project data is available.
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { Promise<void> }
   */
  fetchWorkspaceLevelProjectEntities = (workspaceSlug: string, projectId: string): void => {
    void this.store.projectRoot.project.fetchProject(projectId);
  };

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

  // actions
  /**
   * @description Fetches the user's project information
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { Promise<TProjectMembership | undefined> }
   */
  fetchUserProjectInfo = async (workspaceSlug: string, projectId: string): Promise<TProjectMembership> => {
    try {
      const response = await projectMemberService.projectMemberMe(workspaceSlug, projectId);
      if (response) {
        runInAction(() => {
          set(this.projectUserInfo, [workspaceSlug, projectId], response);
          set(this.workspaceProjectsPermissions, [workspaceSlug, projectId], response.role);
        });
      }
      return response;
    } catch (error) {
      console.error("Error fetching user project information", error);
      throw error;
    }
  };

  /**
   * @description Fetches the user's project permissions
   * @param { string } workspaceSlug
   * @returns { Promise<IUserProjectsRole | undefined> }
   */
  fetchUserProjectPermissions = async (workspaceSlug: string): Promise<IUserProjectsRole> => {
    try {
      const response = await this.workspaceService.getWorkspaceUserProjectsRole(workspaceSlug);
      runInAction(() => {
        set(this.workspaceProjectsPermissions, [workspaceSlug], response);
      });
      return response;
    } catch (error) {
      console.error("Error fetching user project permissions", error);
      throw error;
    }
  };

  /**
   * @description Joins a project
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { Promise<void> }
   */
  joinProject = async (workspaceSlug: string, projectId: string): Promise<void> => {
    try {
      const response = await this.userService.joinProject(workspaceSlug, [projectId]);
      const projectMemberRole = this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) ?? EUserPermissions.MEMBER;
      if (response) {
        runInAction(() => {
          set(this.workspaceProjectsPermissions, [workspaceSlug, projectId], projectMemberRole);
        });
        void this.fetchWorkspaceLevelProjectEntities(workspaceSlug, projectId);
      }
    } catch (error) {
      console.error("Error user joining the project", error);
      throw error;
    }
  };

  /**
   * @description Leaves a project
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { Promise<void> }
   */
  leaveProject = async (workspaceSlug: string, projectId: string): Promise<void> => {
    try {
      await this.userService.leaveProject(workspaceSlug, projectId);
      runInAction(() => {
        unset(this.workspaceProjectsPermissions, [workspaceSlug, projectId]);
        unset(this.projectUserInfo, [workspaceSlug, projectId]);
      });
    } catch (error) {
      console.error("Error user leaving the project", error);
      throw error;
    }
  };
}
