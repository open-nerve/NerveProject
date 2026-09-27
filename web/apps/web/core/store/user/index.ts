/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction, computed } from "mobx";
// nerve imports
import type {
  ApiClient,
  ChangePasswordRequest,
  LoginRequest,
  RegisterRequest,
  User,
  UserUpdate,
} from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import type { TUserPermissions } from "@nerve/types";
// lib
import { tokenManager } from "@/lib/auth/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
// store
import type { RootStore } from "@/store/root.store";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
// services
import { AuthService } from "@/services/auth.service";
import { UserService } from "@/services/user.service";
// stores
import type { IUserProfileStore } from "@/store/user/profile.store";
import { ProfileStore } from "@/store/user/profile.store";
// local imports
import type { IUserSettingsStore } from "./settings.store";
import { UserSettingsStore } from "./settings.store";

export interface IUserStore {
  // observables
  isLoading: boolean;
  data: User | undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
  // actions
  fetchCurrentUser: () => Promise<User | undefined>;
  updateCurrentUser: (data: UserUpdate) => Promise<User>;
  deactivateAccount: () => Promise<void>;
  changePassword: (payload: ChangePasswordRequest) => Promise<void>;
  signIn: (credentials: LoginRequest) => Promise<void>;
  signUp: (credentials: RegisterRequest) => Promise<void>;
  signOut: () => Promise<void>;
  // computed
  canPerformAnyCreateAction: boolean;
  projectsWithCreatePermissions: { [projectId: string]: number } | null;
}

export class UserStore implements IUserStore {
  // observables
  isLoading: boolean = false;
  data: User | undefined = undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
  // service
  userService: UserService;
  authService: AuthService;

  constructor(
    private store: RootStore,
    api: ApiClient
  ) {
    // stores
    this.userProfile = new ProfileStore(store, api);
    this.userSettings = new UserSettingsStore(api);
    this.permission = new UserPermissionStore(store, api);
    // service
    this.userService = new UserService(api);
    this.authService = new AuthService();
    // observables
    makeObservable(this, {
      // observables
      isLoading: observable.ref,
      // model observables
      data: observable,
      userProfile: observable,
      userSettings: observable,
      permission: observable,
      // actions
      fetchCurrentUser: action,
      updateCurrentUser: action,
      deactivateAccount: action,
      changePassword: action,
      signIn: action,
      signUp: action,
      signOut: action,
      // computed
      canPerformAnyCreateAction: computed,
      projectsWithCreatePermissions: computed,
    });
  }

  /**
   * @description fetches the account and its profile, once the session is decided: without one it asks
   * nerve nothing (M2 design 7.1). The workspaces come with M3 (M2 design 3.1). A change of session while
   * they load is no failure: the new session has a RootStore of its own (store-context.tsx), and
   * AuthenticationWrapper fetches the new session's account through it.
   * @returns {Promise<User | undefined>}
   */
  fetchCurrentUser = async (): Promise<User | undefined> => {
    await tokenManager.start();
    if (tokenManager.state.status !== "signed-in") return undefined;
    runInAction(() => {
      this.isLoading = true;
    });
    try {
      const [user] = await Promise.all([this.userService.currentUser(), this.userProfile.fetchUserProfile()]);
      runInAction(() => {
        this.data = user;
      });
      return user;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    } finally {
      runInAction(() => {
        this.isLoading = false;
      });
    }
  };

  /**
   * @description updates the account's names or time zone
   * @returns {Promise<User>}
   */
  updateCurrentUser = async (data: UserUpdate): Promise<User> => {
    const user = await this.userService.updateCurrentUser(data);
    runInAction(() => {
      this.data = user;
    });
    return user;
  };

  changePassword = async (payload: ChangePasswordRequest): Promise<void> => {
    await this.userService.changePassword(payload);
  };

  /**
   * @description deactivates the account; nerve ends all its sessions, and this browser forgets its own:
   * the session the account was deactivated in, read before the request, since the tab may follow another
   * tab's sign-in while the request is out
   * @returns {Promise<void>}
   */
  deactivateAccount = async (): Promise<void> => {
    const loginId = tokenManager.state.loginId;
    await this.userService.deactivate();
    await tokenManager.endSession(loginId);
  };

  /**
   * @description signs in; the token manager keeps the session, and AuthenticationWrapper fetches the
   * account and moves on (M2 design 7.3). Fails, with nothing kept, when nerve refuses.
   * @returns {Promise<void>}
   */
  signIn = async (credentials: LoginRequest): Promise<void> => {
    await tokenManager.signIn(await this.authService.login(credentials));
  };

  /**
   * @description creates the account and signs it in, as signIn does
   * @returns {Promise<void>}
   */
  signUp = async (credentials: RegisterRequest): Promise<void> => {
    await tokenManager.signIn(await this.authService.register(credentials));
  };

  /**
   * @description signs out this browser's session; once it ends, a new RootStore without a session takes
   * over (store-context.tsx)
   * @returns {Promise<void>}
   */
  signOut = async (): Promise<void> => {
    await tokenManager.signOut();
  };

  // helper actions
  /**
   * @description fetches the projects with write permissions
   * @returns {{[projectId: string]: number} || null}
   */
  fetchProjectsWithCreatePermissions = (): { [key: string]: TUserPermissions } => {
    const { workspaceSlug } = this.store.router;

    const allWorkspaceProjectRoles = this.permission.getProjectRolesByWorkspaceSlug(workspaceSlug || "");

    const userPermissions =
      (allWorkspaceProjectRoles &&
        Object.keys(allWorkspaceProjectRoles)
          .filter((key) => allWorkspaceProjectRoles[key] >= EUserPermissions.MEMBER)
          .reduce(
            (res: { [projectId: string]: number }, key: string) => ((res[key] = allWorkspaceProjectRoles[key]), res),
            {}
          )) ||
      null;

    return userPermissions;
  };

  /**
   * @description returns projects where user has permissions
   * @returns {{[projectId: string]: number} || null}
   */
  get projectsWithCreatePermissions() {
    return this.fetchProjectsWithCreatePermissions();
  }

  /**
   * @description returns true if user has permissions to write in any project
   * @returns {boolean}
   */
  get canPerformAnyCreateAction() {
    const filteredProjects = this.fetchProjectsWithCreatePermissions();
    return filteredProjects ? Object.keys(filteredProjects).length > 0 : false;
  }
}
