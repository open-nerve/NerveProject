/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { set, sortBy } from "lodash-es";
import { action, computed, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type {
  ApiClient,
  WorkspaceInvitation,
  WorkspaceInvitationUpdate,
  WorkspaceInvitationsCreate,
  WorkspaceMember,
  WorkspaceMemberUpdate,
} from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspaceInvitationsService } from "@/services/workspace/workspace-invitations.service";
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
  getWorkspaceInvitationDetails: (invitationId: string) => WorkspaceInvitation | null;
  // fetch actions
  fetchWorkspaceMembers: (workspaceSlug: string) => Promise<WorkspaceMember[] | undefined>;
  fetchWorkspaceMemberInvitations: (workspaceSlug: string) => Promise<WorkspaceInvitation[] | undefined>;
  // crud actions
  updateMember: (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate) => Promise<WorkspaceMember>;
  removeMemberFromWorkspace: (workspaceSlug: string, userId: string) => Promise<void>;
  // invite actions
  inviteMembersToWorkspace: (workspaceSlug: string, data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
  updateMemberInvitation: (
    workspaceSlug: string,
    invitationId: string,
    data: WorkspaceInvitationUpdate
  ) => Promise<WorkspaceInvitation>;
  deleteMemberInvitation: (workspaceSlug: string, invitationId: string) => Promise<void>;
  isUserSuspended: (userId: string, workspaceSlug: string | undefined) => boolean;
}

/**
 * The members and the invitations of the workspaces of a session (M3 design 7.3): its services send with the
 * session's client, which the RootStore of the session hands down. Changes go one at a time (v0 design 7.7);
 * fetches do not queue.
 */
export class WorkspaceMemberStore implements IWorkspaceMemberStore {
  // observables
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>> = {};
  /** Each workspace's invitations, pending and declined, newest first: an admin's to fetch, whom nerve shows them. */
  workspaceMemberInvitations: Record<string, WorkspaceInvitation[]> = {};
  // filters store
  filtersStore: IWorkspaceMemberFiltersStore;
  // stores
  routerStore: IRouterStore;
  userStore: IUserStore;
  /** The users the stores read, which the members' profiles join. */
  memberRoot: Pick<IMemberRootStore, "memberMap">;
  // services
  private readonly service: WorkspaceMembersService;
  private readonly invitationsService: WorkspaceInvitationsService;
  /** The changes of the memberships and the invitations, sent one at a time. */
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
    this.service = new WorkspaceMembersService(api);
    this.invitationsService = new WorkspaceInvitationsService(api);
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
   * @description fetches a workspace's invitations and gives them: an admin's to fetch, as nerve refuses anyone
   * else. A change of session while they load is no failure: the new session's store fetches its own
   * (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspaceInvitation[] | undefined>}
   */
  fetchWorkspaceMemberInvitations = async (workspaceSlug: string): Promise<WorkspaceInvitation[] | undefined> => {
    try {
      const invitations = await this.invitationsService.list(workspaceSlug);
      runInAction(() => {
        this.workspaceMemberInvitations[workspaceSlug] = invitations;
      });
      return invitations;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description invites the addresses data lists, all of them or none; the new invitations go first in the list
   * the store has, until the next fetch puts them in nerve's order. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<WorkspaceInvitation[]>}
   */
  inviteMembersToWorkspace = (
    workspaceSlug: string,
    data: WorkspaceInvitationsCreate
  ): Promise<WorkspaceInvitation[]> =>
    this.changes(async () => {
      const created = await this.invitationsService.create(workspaceSlug, data);
      this.changeInvitations(workspaceSlug, (invitations) => [...created, ...invitations]);
      return created;
    });

  /**
   * @description changes an invitation's role; the store then has nerve's answer. Fails, changing nothing, when
   * nerve refuses (a declined invitation, 409).
   * @returns {Promise<WorkspaceInvitation>}
   */
  updateMemberInvitation = (
    workspaceSlug: string,
    invitationId: string,
    data: WorkspaceInvitationUpdate
  ): Promise<WorkspaceInvitation> =>
    this.changes(async () => {
      const changed = await this.invitationsService.update(invitationId, data);
      this.changeInvitations(workspaceSlug, (invitations) =>
        invitations.map((i) => (i.id === changed.id ? changed : i))
      );
      return changed;
    });

  /**
   * @description deletes an invitation, which then leaves the list; fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteMemberInvitation = (workspaceSlug: string, invitationId: string): Promise<void> =>
    this.changes(async () => {
      await this.invitationsService.delete(invitationId);
      this.changeInvitations(workspaceSlug, (invitations) => invitations.filter((i) => i.id !== invitationId));
    });

  /** The workspace's invitations as change makes them, when the store has fetched them: it makes up no list. */
  private changeInvitations(
    workspaceSlug: string,
    change: (invitations: WorkspaceInvitation[]) => WorkspaceInvitation[]
  ): void {
    runInAction(() => {
      const listed = this.workspaceMemberInvitations[workspaceSlug];
      if (listed) this.workspaceMemberInvitations[workspaceSlug] = change(listed);
    });
  }

  isUserSuspended = computedFn((userId: string, workspaceSlug: string | undefined) => {
    if (!workspaceSlug) return false;
    const workspaceMember = this.workspaceMemberMap?.[workspaceSlug]?.[userId];
    return workspaceMember?.is_active === false;
  });
}
