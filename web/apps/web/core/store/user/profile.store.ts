/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, Profile, ProfileUpdate, Theme } from "@nerve/api-client";
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
  /** The number of the last update sent, and of the one whose answer the profile holds (updateUserProfile). */
  private updatesSent = 0;
  private updateWritten = 0;

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
   * The profile becomes nerve's answer, never the change asked for: a refused change leaves it as it was. The
   * updates are numbered as they are sent, and an answer older than the one the profile holds is dropped: this
   * assumes nerve applies the updates in the order they were sent, as the PAT store assumes of its requests, so
   * the older answer is an older profile. Only answers count: when the newer update fails, the older one's
   * answer still writes.
   * @returns {Promise<Profile>}
   */
  updateUserProfile = async (data: ProfileUpdate): Promise<Profile> => {
    const update = ++this.updatesSent;
    const profile = await this.userService.updateCurrentUserProfile(data);
    if (update > this.updateWritten) {
      this.updateWritten = update;
      runInAction(() => {
        this.data = profile;
      });
    }
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
