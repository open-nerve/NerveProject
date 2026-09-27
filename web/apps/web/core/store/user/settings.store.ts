/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient } from "@nerve/api-client";
import type { IUserSettings } from "@nerve/types";
// services
import { UserService } from "@/services/user.service";

export interface IUserSettingsStore {
  // observables
  data: IUserSettings;
  sidebarCollapsed: boolean;
  // actions
  fetchCurrentUserSettings: (bustCache?: boolean) => Promise<IUserSettings>;
  toggleSidebar: (collapsed?: boolean) => void;
}

export class UserSettingsStore implements IUserSettingsStore {
  // observables
  sidebarCollapsed: boolean = true;
  data: IUserSettings = {
    id: undefined,
    email: undefined,
    workspace: {
      last_workspace_id: undefined,
      last_workspace_slug: undefined,
      last_workspace_name: undefined,
      last_workspace_logo: undefined,
      fallback_workspace_id: undefined,
      fallback_workspace_slug: undefined,
      invites: undefined,
    },
  };
  // services
  userService: UserService;

  constructor(api: ApiClient) {
    makeObservable(this, {
      // observables
      data: observable,
      sidebarCollapsed: observable.ref,
      // actions
      fetchCurrentUserSettings: action,
      toggleSidebar: action,
    });
    // services
    this.userService = new UserService(api);
  }

  // actions
  toggleSidebar = (collapsed?: boolean) => {
    this.sidebarCollapsed = collapsed ?? !this.sidebarCollapsed;
  };

  // actions
  /**
   * @description fetches user profile information
   * @returns {Promise<IUserSettings>}
   */
  fetchCurrentUserSettings = async (bustCache: boolean = false) => {
    const userSettings = await this.userService.currentUserSettings(bustCache);
    runInAction(() => {
      this.data = userSettings;
    });
    return userSettings;
  };
}
