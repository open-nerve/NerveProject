/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, Profile, ProfileUpdate, Theme } from "@nerve/api-client";
import { setLanguage } from "@nerve/i18n";
// services
import { UserService } from "@/services/user.service";
// store
import type { RootStore } from "../root.store";

export interface IUserProfileStore {
  // observables
  data: Profile | undefined;
  // actions
  fetchUserProfile: () => Promise<Profile>;
  updateUserProfile: (data: ProfileUpdate) => Promise<Profile>;
  finishUserOnboarding: () => Promise<void>;
  updateTourCompleted: () => Promise<Profile>;
  updateUserTheme: (theme: Theme) => Promise<Profile>;
}

export class ProfileStore implements IUserProfileStore {
  data: Profile | undefined = undefined;

  // services
  userService: UserService;

  constructor(
    public store: RootStore,
    api: ApiClient
  ) {
    makeObservable(this, {
      // observables
      data: observable,
      // actions
      fetchUserProfile: action,
      updateUserProfile: action,
      finishUserOnboarding: action,
      updateTourCompleted: action,
      updateUserTheme: action,
    });
    // services
    this.userService = new UserService(api);
  }

  /**
   * @description fetches the account's profile, and shows the app in its language
   * @returns {Promise<Profile>}
   */
  fetchUserProfile = async (): Promise<Profile> => {
    const profile = await this.userService.getCurrentUserProfile();
    runInAction(() => {
      this.data = profile;
    });
    void setLanguage(profile.language);
    return profile;
  };

  /**
   * @description changes the given fields of the profile (onboarding_step key by key); fails when nerve refuses
   * @returns {Promise<Profile>}
   */
  updateUserProfile = async (data: ProfileUpdate): Promise<Profile> => {
    if (data.language) void setLanguage(data.language);
    const profile = await this.userService.updateCurrentUserProfile(data);
    runInAction(() => {
      this.data = profile;
    });
    return profile;
  };

  /**
   * @description finishes the onboarding in one change of the profile
   * @returns {Promise<void>}
   */
  finishUserOnboarding = async (): Promise<void> => {
    const firstWorkspace = Object.values(this.store.workspaceRoot.workspaces ?? {})[0];
    await this.updateUserProfile({
      onboarding_step: {
        profile_complete: true,
        workspace_join: true,
        workspace_create: true,
        workspace_invite: true,
      },
      is_onboarded: true,
      ...(firstWorkspace ? { last_workspace_id: firstWorkspace.id } : {}),
    });
  };

  /**
   * @description marks the product tour as seen
   * @returns {Promise<Profile>}
   */
  updateTourCompleted = (): Promise<Profile> => this.updateUserProfile({ is_tour_completed: true });

  /**
   * @description changes the theme
   * @returns {Promise<Profile>}
   */
  updateUserTheme = (theme: Theme): Promise<Profile> => this.updateUserProfile({ theme });
}
