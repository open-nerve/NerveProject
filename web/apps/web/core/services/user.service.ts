/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type { ApiClient, ChangePasswordRequest, Profile, ProfileUpdate, User, UserUpdate } from "@nerve/api-client";
import type { TIssuesResponse } from "@nerve/types";
import { unwrap } from "@/lib/api-error";
import { APIService } from "@/services/api.service";

export class UserService extends APIService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {
    super();
  }

  async currentUser(): Promise<User> {
    return unwrap(await this.api.GET("/api/v0/me"));
  }

  async updateCurrentUser(data: UserUpdate): Promise<User> {
    return unwrap(await this.api.PATCH("/api/v0/me", { body: data }));
  }

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    unwrap(await this.api.POST("/api/v0/me/change-password", { body: data }));
  }

  async deactivate(): Promise<void> {
    unwrap(await this.api.POST("/api/v0/me/deactivate"));
  }

  async getCurrentUserProfile(): Promise<Profile> {
    return unwrap(await this.api.GET("/api/v0/me/profile"));
  }

  async updateCurrentUserProfile(data: ProfileUpdate): Promise<Profile> {
    return unwrap(await this.api.PATCH("/api/v0/me/profile", { body: data }));
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
}
