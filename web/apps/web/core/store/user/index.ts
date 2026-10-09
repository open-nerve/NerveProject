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
import { oneAtATime } from "@/lib/one-at-a-time";
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
import type { IApiTokenStore } from "./api-token.store";
import { ApiTokenStore } from "./api-token.store";

export interface IUserStore {
  // observables
  data: User | undefined;
  // store observables
  userProfile: IUserProfileStore;
  permission: IUserPermissionStore;
  apiTokens: IApiTokenStore;
  // actions
  fetchCurrentUser: () => Promise<User | undefined>;
  updateCurrentUser: (data: UserUpdate) => Promise<User>;
  deactivateAccount: () => Promise<boolean>;
  changePassword: (payload: ChangePasswordRequest) => Promise<void>;
  signIn: (credentials: LoginRequest) => Promise<void>;
  signUp: (credentials: RegisterRequest) => Promise<void>;
  signOut: () => Promise<void>;
  // computed
  canPerformAnyCreateAction: boolean;
  projectsWithCreatePermissions: { [projectId: string]: TUserPermissions };
}

export class UserStore implements IUserStore {
  // observables
  data: User | undefined = undefined;
  // store observables
  userProfile: IUserProfileStore;
  permission: IUserPermissionStore;
  apiTokens: IApiTokenStore;
  // service
  userService: UserService;
  authService: AuthService;
  /** The account's changes, sent one at a time (updateCurrentUser). */
  private readonly changes = oneAtATime();

  constructor(
    private store: RootStore,
    api: ApiClient
  ) {
    // stores
    this.userProfile = new ProfileStore(api);
    this.permission = new UserPermissionStore(store);
    this.apiTokens = new ApiTokenStore(api);
    // service
    this.userService = new UserService(api);
    this.authService = new AuthService();
    // observables
    makeObservable(this, {
      // model observables
      data: observable,
      userProfile: observable,
      permission: observable,
      apiTokens: observable,
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
   * nerve nothing (M2 design 7.1). The workspaces are not fetched here: useWorkspacesFetch fetches them, where a
   * page needs them (M3 design 7.1). A change of session while the account and profile load is no failure: the new
   * session has a RootStore of its own (store-context.tsx), and AuthenticationWrapper fetches the new session's account
   * through it.
   * @returns {Promise<User | undefined>}
   */
  fetchCurrentUser = async (): Promise<User | undefined> => {
    await tokenManager.start();
    if (tokenManager.state.status !== "signed-in") return undefined;
    try {
      const [user] = await Promise.all([this.userService.currentUser(), this.userProfile.fetchUserProfile()]);
      runInAction(() => {
        this.data = user;
      });
      return user;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description updates the account's names or time zone; fails, writing nothing, when nerve refuses. An update
   * is sent once the one before it is answered, or has failed: nerve applies them in the order they were made, and
   * the answer to the last one is the account nerve holds.
   * @returns {Promise<User>}
   */
  updateCurrentUser = (data: UserUpdate): Promise<User> =>
    this.changes(async () => {
      const user = await this.userService.updateCurrentUser(data);
      runInAction(() => {
        this.data = user;
      });
      return user;
    });

  changePassword = async (payload: ChangePasswordRequest): Promise<void> => {
    await this.userService.changePassword(payload);
  };

  /**
   * @description deactivates the account; nerve ends all its sessions, and this browser forgets its own:
   * the session the account was deactivated in, read before the request, since the tab may follow another
   * tab's sign-in while the request is out. Resolves whether it ended that session: false when the tab's record was
   * no longer that session's (another tab signed out, or moved this one to another account, whose page it then is)
   * @returns {Promise<boolean>}
   */
  deactivateAccount = async (): Promise<boolean> => {
    const loginId = tokenManager.state.loginId;
    await this.userService.deactivate();
    return tokenManager.endSession(loginId);
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

  /**
   * @description the current workspace's projects, not archived, in which the caller may create (his role a member's
   * or an admin's), each with his role; read from the stores, nothing is fetched
   */
  get projectsWithCreatePermissions() {
    const workspaceSlug = this.store.router.workspaceSlug ?? "";
    const roles: { [projectId: string]: TUserPermissions } = {};
    for (const projectId of this.store.projectRoot.project.joinedProjectIds) {
      const role = this.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
      if (role !== undefined && role >= EUserPermissions.MEMBER) roles[projectId] = role;
    }
    return roles;
  }

  /** @description whether the caller may create in any project of the current workspace */
  get canPerformAnyCreateAction() {
    return Object.keys(this.projectsWithCreatePermissions).length > 0;
  }
}
