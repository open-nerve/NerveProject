/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { set, sortBy } from "lodash-es";
import { action, computed, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, WorkspaceMember, WorkspaceMemberUpdate } from "@nerve/api-client";
import type { IWorkspaceBulkInviteFormData, IWorkspaceMemberInvitation } from "@nerve/types";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspaceService } from "@/services/workspace.service";
import { WorkspaceMembersService } from "@/services/workspace/workspace-members.service";
// types
import type { IRouterStore } from "@/store/router.store";
import type { IUserStore } from "@/store/user";
// store
import type { IMemberRootStore } from "../index.ts";
import type { IWorkspaceMemberFiltersStore } from "./workspace-member-filters.store";
import { WorkspaceMemberFiltersStore } from "./workspace-member-filters.store";
import type { RootStore } from "@/store/root.store";

export interface IWorkspaceMemberStore {
  // observables
  /** Each workspace's memberships by the member's account id, those that ended too (is_active false). */
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>>;
  workspaceMemberInvitations: Record<string, IWorkspaceMemberInvitation[]>;
  // filters store
  filtersStore: IWorkspaceMemberFiltersStore;
  // computed
  workspaceMemberIds: string[] | null;
  workspaceMemberInvitationIds: string[] | null;
  memberMap: Record<string, WorkspaceMember> | null;
  // computed actions
  getWorkspaceMemberIds: (workspaceSlug: string) => string[];
  getFilteredWorkspaceMemberIds: (workspaceSlug: string) => string[];
  getSearchedWorkspaceMemberIds: (searchQuery: string) => string[] | null;
  getSearchedWorkspaceInvitationIds: (searchQuery: string) => string[] | null;
  getWorkspaceMemberDetails: (userId: string) => WorkspaceMember | null;
  getWorkspaceInvitationDetails: (invitationId: string) => IWorkspaceMemberInvitation | null;
  // fetch actions
  fetchWorkspaceMembers: (workspaceSlug: string) => Promise<WorkspaceMember[] | undefined>;
  fetchWorkspaceMemberInvitations: (workspaceSlug: string) => Promise<IWorkspaceMemberInvitation[]>;
  // crud actions
  updateMember: (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate) => Promise<WorkspaceMember>;
  removeMemberFromWorkspace: (workspaceSlug: string, userId: string) => Promise<void>;
  // invite actions
  inviteMembersToWorkspace: (workspaceSlug: string, data: IWorkspaceBulkInviteFormData) => Promise<void>;
  updateMemberInvitation: (
    workspaceSlug: string,
    invitationId: string,
    data: Partial<IWorkspaceMemberInvitation>
  ) => Promise<void>;
  deleteMemberInvitation: (workspaceSlug: string, invitationId: string) => Promise<void>;
  isUserSuspended: (userId: string, workspaceSlug: string | undefined) => boolean;
}

/**
 * The members of the workspaces of a session (M3 design 7.3): its service sends with the session's client, which
 * the RootStore of the session hands down. Changes go one at a time (v0 design 7.7); fetches do not queue.
 */
export class WorkspaceMemberStore implements IWorkspaceMemberStore {
  // observables
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>> = {};
  workspaceMemberInvitations: Record<string, IWorkspaceMemberInvitation[]> = {}; // { workspaceSlug: [invitations] }
  // filters store
  filtersStore: IWorkspaceMemberFiltersStore;
  // stores
  routerStore: IRouterStore;
  userStore: IUserStore;
  /** The users the stores read, which the members' profiles join. */
  memberRoot: Pick<IMemberRootStore, "memberMap">;
  // services
  workspaceService;
  private readonly service: WorkspaceMembersService;
  /** The changes of the memberships, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(_memberRoot: Pick<IMemberRootStore, "memberMap">, _rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // observables
      workspaceMemberMap: observable,
      workspaceMemberInvitations: observable,
      // computed
      workspaceMemberIds: computed,
      workspaceMemberInvitationIds: computed,
      memberMap: computed,
      // actions
      fetchWorkspaceMembers: action,
      updateMember: action,
      removeMemberFromWorkspace: action,
      fetchWorkspaceMemberInvitations: action,
      updateMemberInvitation: action,
      deleteMemberInvitation: action,
    });
    // initialize filters store
    this.filtersStore = new WorkspaceMemberFiltersStore();
    // root store
    this.routerStore = _rootStore.router;
    this.userStore = _rootStore.user;
    this.memberRoot = _memberRoot;
    // services
    this.workspaceService = new WorkspaceService();
    this.service = new WorkspaceMembersService(api);
  }

  /**
   * @description get the list of all the user ids of all the members of the current workspace
   */
  get workspaceMemberIds() {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;

    return this.getWorkspaceMemberIds(workspaceSlug);
  }

  get memberMap() {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.workspaceMemberMap?.[workspaceSlug] ?? {};
  }

  get workspaceMemberInvitationIds() {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.workspaceMemberInvitations?.[workspaceSlug]?.map((inv) => inv.id);
  }

  getWorkspaceMemberIds = computedFn((workspaceSlug: string) => {
    const members = sortBy(Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {}), [
      (m) => m.member.id !== this.userStore?.data?.id,
      (m) => m.member.display_name.toLowerCase(),
    ]);
    return members.map((m) => m.member.id);
  });

  /**
   * @description get the filtered and sorted list of all the user ids of all the members of the workspace
   * @param workspaceSlug
   */
  getFilteredWorkspaceMemberIds = computedFn((workspaceSlug: string) =>
    this.filtersStore.getFilteredMemberIds(
      Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {}),
      this.memberRoot?.memberMap || {},
      (member) => member.member.id
    )
  );

  /**
   * @description get the list of all the user ids that match the search query of all the members of the current workspace
   * @param searchQuery
   */
  getSearchedWorkspaceMemberIds = computedFn((searchQuery: string) => {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    const filteredMemberIds = this.getFilteredWorkspaceMemberIds(workspaceSlug);
    if (!filteredMemberIds) return null;
    const searchedWorkspaceMemberIds = filteredMemberIds.filter((userId) => {
      const memberDetails = this.getWorkspaceMemberDetails(userId);
      if (!memberDetails) return false;
      const memberSearchQuery = `${memberDetails.member.first_name} ${memberDetails.member.last_name} ${
        memberDetails.member?.display_name
      } ${memberDetails.member.email ?? ""}`;
      return memberSearchQuery.toLowerCase()?.includes(searchQuery.toLowerCase());
    });
    return searchedWorkspaceMemberIds;
  });

  /**
   * @description get the list of all the invitation ids that match the search query of all the member invitations of the current workspace
   * @param searchQuery
   */
  getSearchedWorkspaceInvitationIds = computedFn((searchQuery: string) => {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    const workspaceMemberInvitationIds = this.workspaceMemberInvitationIds;
    if (!workspaceMemberInvitationIds) return null;
    const searchedWorkspaceMemberInvitationIds = workspaceMemberInvitationIds.filter((invitationId) => {
      const invitationDetails = this.getWorkspaceInvitationDetails(invitationId);
      if (!invitationDetails) return false;
      const invitationSearchQuery = `${invitationDetails.email}`;
      return invitationSearchQuery.toLowerCase()?.includes(searchQuery.toLowerCase());
    });
    return searchedWorkspaceMemberInvitationIds;
  });

  /**
   * @description get the details of a workspace member
   * @param userId
   */
  getWorkspaceMemberDetails = computedFn((userId: string) => {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.workspaceMemberMap?.[workspaceSlug]?.[userId] ?? null;
  });

  /**
   * @description get the details of a workspace member invitation
   * @param workspaceSlug
   * @param memberId
   */
  getWorkspaceInvitationDetails = computedFn((invitationId: string) => {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    const invitationsList = this.workspaceMemberInvitations?.[workspaceSlug];
    if (!invitationsList) return null;

    const invitation = invitationsList.find((inv) => inv.id === invitationId);
    return invitation ?? null;
  });

  /**
   * @description fetches a workspace's memberships, those that ended too, and gives them; each member's profile
   * joins the users the other stores read (memberRoot.memberMap). A change of session while they load is no
   * failure: the new session's store fetches its own (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspaceMember[] | undefined>}
   */
  fetchWorkspaceMembers = async (workspaceSlug: string): Promise<WorkspaceMember[] | undefined> => {
    try {
      const memberships = await this.service.list(workspaceSlug);
      runInAction(() => {
        for (const membership of memberships) set(this.memberRoot.memberMap, membership.member.id, membership.member);
        this.workspaceMemberMap[workspaceSlug] = Object.fromEntries(memberships.map((m) => [m.member.id, m]));
      });
      return memberships;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description changes the role of a member of the workspace; the store then has nerve's answer. Fails,
   * changing nothing, when nerve refuses or the store has no membership of his.
   * @returns {Promise<WorkspaceMember>}
   */
  updateMember = (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate): Promise<WorkspaceMember> =>
    this.changes(async () => {
      const membership = await this.service.update(this.membership(workspaceSlug, userId).id, data);
      runInAction(() => {
        set(this.workspaceMemberMap, [workspaceSlug, userId], membership);
      });
      return membership;
    });

  /**
   * @description ends the membership of a member of the workspace, which the store then keeps as ended
   * (is_active false), as nerve lists it. Fails, changing nothing, when nerve refuses or the store has no
   * membership of his.
   * @returns {Promise<void>}
   */
  removeMemberFromWorkspace = (workspaceSlug: string, userId: string): Promise<void> =>
    this.changes(async () => {
      const membership = this.membership(workspaceSlug, userId);
      await this.service.remove(membership.id);
      runInAction(() => {
        set(this.workspaceMemberMap, [workspaceSlug, userId], { ...membership, is_active: false });
      });
    });

  /** The membership of the member userId names in the workspace, as the store has it; fails when it has none. */
  private membership(workspaceSlug: string, userId: string): WorkspaceMember {
    const membership = this.workspaceMemberMap[workspaceSlug]?.[userId];
    if (!membership) throw new Error("Member not found");
    return membership;
  }

  /**
   * @description fetch all the member invitations of a workspace
   * @param workspaceSlug
   */
  fetchWorkspaceMemberInvitations = async (workspaceSlug: string) =>
    await this.workspaceService.workspaceInvitations(workspaceSlug).then((response) => {
      runInAction(() => {
        set(this.workspaceMemberInvitations, workspaceSlug, response);
      });
      return response;
    });

  /**
   * @description bulk invite members to a workspace
   * @param workspaceSlug
   * @param data
   */
  inviteMembersToWorkspace = async (workspaceSlug: string, data: IWorkspaceBulkInviteFormData) => {
    const response = await this.workspaceService.inviteWorkspace(workspaceSlug, data);
    await this.fetchWorkspaceMemberInvitations(workspaceSlug);
    return response;
  };

  /**
   * @description update the role of a member invitation
   * @param workspaceSlug
   * @param invitationId
   * @param data
   */
  updateMemberInvitation = async (
    workspaceSlug: string,
    invitationId: string,
    data: Partial<IWorkspaceMemberInvitation>
  ) => {
    const originalMemberInvitations = [...(this.workspaceMemberInvitations?.[workspaceSlug] ?? [])]; // in case of error, we will revert back to original members
    try {
      const memberInvitations = originalMemberInvitations?.map((invitation) => ({
        ...invitation,
        ...(invitation.id === invitationId && data),
      }));
      // optimistic update
      runInAction(() => {
        set(this.workspaceMemberInvitations, workspaceSlug, memberInvitations);
      });
      await this.workspaceService.updateWorkspaceInvitation(workspaceSlug, invitationId, data);
    } catch (error) {
      // revert back to original members in case of error
      runInAction(() => {
        set(this.workspaceMemberInvitations, workspaceSlug, originalMemberInvitations);
      });
      throw error;
    }
  };

  /**
   * @description delete a member invitation
   * @param workspaceSlug
   * @param memberId
   */
  deleteMemberInvitation = async (workspaceSlug: string, invitationId: string) =>
    // oxlint-disable-next-line promise/always-return
    await this.workspaceService.deleteWorkspaceInvitations(workspaceSlug, invitationId).then(() => {
      runInAction(() => {
        this.workspaceMemberInvitations[workspaceSlug] = this.workspaceMemberInvitations[workspaceSlug].filter(
          (inv) => inv.id !== invitationId
        );
      });
    });

  isUserSuspended = computedFn((userId: string, workspaceSlug: string | undefined) => {
    if (!workspaceSlug) return false;
    const workspaceMember = this.workspaceMemberMap?.[workspaceSlug]?.[userId];
    return workspaceMember?.is_active === false;
  });
}
