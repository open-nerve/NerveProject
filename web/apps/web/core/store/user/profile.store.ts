/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, Profile, ProfileUpdate, Theme } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { UserService } from "@/services/user.service";

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
  /** The profile's changes, sent one at a time (updateUserProfile). */
  private readonly changes = oneAtATime();

  // services
  userService: UserService;

  constructor(api: ApiClient) {
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
   * @description fetches the account's profile. The page's language follows the profile of the tab's session now
   * (StoreWrapper): a store sets no page state, so a retired session's store cannot reach the page.
   * @returns {Promise<Profile>}
   */
  fetchUserProfile = async (): Promise<Profile> => {
    const profile = await this.userService.getCurrentUserProfile();
    runInAction(() => {
      this.data = profile;
    });
    return profile;
  };

  /**
   * @description changes the given fields of the profile (onboarding_step key by key); fails when nerve refuses.
   * The profile becomes nerve's answer, never the change asked for: a refused change leaves it as it was. A change
   * is sent once the one before it is answered, or has failed: nerve applies them in the order they were made, and
   * the answer to the last one is the profile nerve holds.
   * @returns {Promise<Profile>}
   */
  updateUserProfile = (data: ProfileUpdate): Promise<Profile> =>
    this.changes(async () => {
      const profile = await this.userService.updateCurrentUserProfile(data);
      runInAction(() => {
        this.data = profile;
      });
      return profile;
    });

  /**
   * @description finishes the onboarding in one change of the profile
   * @returns {Promise<void>}
   */
  finishUserOnboarding = async (): Promise<void> => {
    await this.updateUserProfile({
      onboarding_step: {
        profile_complete: true,
        workspace_join: true,
        workspace_create: true,
        workspace_invite: true,
      },
      is_onboarded: true,
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
