/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type { ChangePasswordRequest, Profile, ProfileUpdate, User, UserUpdate } from "@nerve/api-client";
import type { IUserSettings, TIssuesResponse } from "@nerve/types";
import { unwrap } from "@/lib/api-error";
import { api } from "@/lib/auth/api-client";
import { APIService } from "@/services/api.service";

export class UserService extends APIService {
  async currentUser(): Promise<User> {
    return unwrap(await api.GET("/api/v0/me"));
  }

  async updateCurrentUser(data: UserUpdate): Promise<User> {
    return unwrap(await api.PATCH("/api/v0/me", { body: data }));
  }

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    unwrap(await api.POST("/api/v0/me/change-password", { body: data }));
  }

  async deactivate(): Promise<void> {
    unwrap(await api.POST("/api/v0/me/deactivate"));
  }

  async getCurrentUserProfile(): Promise<Profile> {
    return unwrap(await api.GET("/api/v0/me/profile"));
  }

  async updateCurrentUserProfile(data: ProfileUpdate): Promise<Profile> {
    return unwrap(await api.PATCH("/api/v0/me/profile", { body: data }));
  }

  async currentUserSettings(bustCache: boolean = false): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.get(url)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async getUserProfileIssues(
    workspaceSlug: string,
    userId: string,
    params: any,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`,
      {
        params,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveWorkspace(workspaceSlug: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<any> {
    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveProject(workspaceSlug: string, projectId: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}

const userService = new UserService();

export default userService;
