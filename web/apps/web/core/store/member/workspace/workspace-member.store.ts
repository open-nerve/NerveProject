/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mapValues, set, sortBy } from "lodash-es";
import { action, computed, makeObservable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type {
  ApiClient,
  Workspace,
  WorkspaceInvitation,
  WorkspaceInvitationUpdate,
  WorkspaceInvitationsCreate,
  WorkspaceMember,
  WorkspaceMemberUpdate,
} from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, dropped, prepended, replaced } from "@/lib/reconciled";
// services
import { WorkspaceInvitationsService } from "@/services/workspace/workspace-invitations.service";
import { WorkspaceMembersService } from "@/services/workspace/workspace-members.service";
// types
import type { IRouterStore } from "@/store/router.store";
import type { IUserStore } from "@/store/user";
import type { IWorkspaceRootStore } from "@/store/workspace";
// store
import type { IMemberRootStore } from "../index.ts";
import type { IWorkspaceMemberFiltersStore } from "./workspace-member-filters.store";
import { WorkspaceMemberFiltersStore } from "./workspace-member-filters.store";
import type { RootStore } from "@/store/root.store";

/** A workspace's memberships by the member's account id, those that ended too (is_active false). */
type Memberships = Record<string, WorkspaceMember>;

export interface IWorkspaceMemberStore {
  // filters store
  filtersStore: IWorkspaceMemberFiltersStore;
  // computed
  workspaceMemberIds: string[] | null;
  workspaceMemberInvitationIds: string[] | null;
  memberMap: Memberships | null;
  // computed actions
  getWorkspaceMemberIds: (workspaceSlug: string) => string[];
  getFilteredWorkspaceMemberIds: (workspaceSlug: string) => string[];
  getSearchedWorkspaceMemberIds: (searchQuery: string) => string[] | null;
  getSearchedWorkspaceInvitationIds: (searchQuery: string) => string[] | null;
  getWorkspaceMemberDetails: (userId: string) => WorkspaceMember | null;
  getWorkspaceInvitationDetails: (invitationId: string) => WorkspaceInvitation | null;
  // fetch actions
  fetchWorkspaceMembers: (workspace: Pick<Workspace, "id" | "slug">) => Promise<Memberships | undefined>;
  fetchWorkspaceMemberInvitations: (
    workspace: Pick<Workspace, "id" | "slug">
  ) => Promise<WorkspaceInvitation[] | undefined>;
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

/** The memberships with membership as nerve gave it. */
const withMembership =
  (membership: WorkspaceMember): Change<Memberships> =>
  (memberships) => ({ ...memberships, [membership.member.id]: membership });

/** The memberships with the one of the member userId names ended, as nerve lists it once he is removed. */
const ended =
  (userId: string): Change<Memberships> =>
  (memberships) =>
    mapValues(memberships, (membership, id) => (id === userId ? { ...membership, is_active: false } : membership));

/**
 * The members and the invitations of the workspaces of a session (M3 design 7.3): its services send with the
 * session's client, which the RootStore of the session hands down. Changes go one at a time (v0 design 7.7);
 * fetches do not queue. A workspace's members and invitations are kept by its id and read by the slug of a workspace
 * on the caller's list, so nothing of a workspace he left or deleted shows, nor of one deleted whose slug names
 * another now.
 */
export class WorkspaceMemberStore implements IWorkspaceMemberStore {
  /** Each workspace's memberships, reconciled between their fetches and the changes nerve confirmed (reconciled.ts). */
  private readonly members = new ReconciledByKey<Memberships>();
  /** Each workspace's invitations, pending and declined, newest first: an admin's to fetch, whom nerve shows them. */
  private readonly invitations = new ReconciledByKey<WorkspaceInvitation[]>();
  // filters store
  filtersStore: IWorkspaceMemberFiltersStore;
  // stores
  routerStore: IRouterStore;
  userStore: IUserStore;
  workspaceRoot: Pick<IWorkspaceRootStore, "getWorkspaceBySlug">;
  /** The users the stores read, which the members' profiles join. */
  memberRoot: Pick<IMemberRootStore, "memberMap">;
  // services
  private readonly service: WorkspaceMembersService;
  private readonly invitationsService: WorkspaceInvitationsService;
  /** The changes of the memberships and the invitations, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(_memberRoot: Pick<IMemberRootStore, "memberMap">, _rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      workspaceMemberIds: computed,
      workspaceMemberInvitationIds: computed,
      memberMap: computed,
      // actions
      fetchWorkspaceMembers: action,
      updateMember: action,
      removeMemberFromWorkspace: action,
      fetchWorkspaceMemberInvitations: action,
      inviteMembersToWorkspace: action,
      updateMemberInvitation: action,
      deleteMemberInvitation: action,
    });
    // initialize filters store
    this.filtersStore = new WorkspaceMemberFiltersStore();
    // root store
    this.routerStore = _rootStore.router;
    this.userStore = _rootStore.user;
    this.workspaceRoot = _rootStore.workspaceRoot;
    this.memberRoot = _memberRoot;
    // services
    this.service = new WorkspaceMembersService(api);
    this.invitationsService = new WorkspaceInvitationsService(api);
  }

  /** The id of the caller's workspace slug names, by his list. */
  private workspaceIdOf(workspaceSlug: string): string | undefined {
    return this.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  }

  /** The memberships of the caller's workspace slug names, once fetched. */
  getMemberships = (workspaceSlug: string): Memberships | undefined =>
    this.members.get(this.workspaceIdOf(workspaceSlug));

  /** The invitations of the caller's workspace slug names, once fetched. */
  getInvitations = (workspaceSlug: string): WorkspaceInvitation[] | undefined =>
    this.invitations.get(this.workspaceIdOf(workspaceSlug));

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
    return this.getMemberships(workspaceSlug) ?? {};
  }

  get workspaceMemberInvitationIds() {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.getInvitations(workspaceSlug)?.map((inv) => inv.id) ?? null;
  }

  getWorkspaceMemberIds = computedFn((workspaceSlug: string) => {
    const members = sortBy(Object.values(this.getMemberships(workspaceSlug) ?? {}), [
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
      Object.values(this.getMemberships(workspaceSlug) ?? {}),
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
    return this.getMemberships(workspaceSlug)?.[userId] ?? null;
  });

  /**
   * @description get the details of a workspace member invitation
   * @param invitationId
   */
  getWorkspaceInvitationDetails = computedFn((invitationId: string) => {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.getInvitations(workspaceSlug)?.find((inv) => inv.id === invitationId) ?? null;
  });

  /**
   * @description fetches a workspace's memberships, those that ended too, and shows them with the changes nerve
   * confirmed meanwhile; each member's profile, as shown, joins the users the other stores read
   * (memberRoot.memberMap). Gives what it shows, or undefined for a fetch a newer one overtook or a change of session
   * cut (Reconciled.fetch).
   * @returns {Promise<Memberships | undefined>}
   */
  fetchWorkspaceMembers = async (workspace: Pick<Workspace, "id" | "slug">): Promise<Memberships | undefined> => {
    const memberships = await this.members.fetch(workspace.id, async () => {
      const listed = await this.service.list(workspace.slug);
      return Object.fromEntries(listed.map((membership) => [membership.member.id, membership]));
    });
    if (memberships) this.shareProfiles(Object.values(memberships));
    return memberships;
  };

  /** The members' profiles of memberships join the users the other stores read. */
  private shareProfiles(memberships: WorkspaceMember[]): void {
    runInAction(() => {
      for (const membership of memberships) set(this.memberRoot.memberMap, membership.member.id, membership.member);
    });
  }

  /**
   * @description changes the role of a member of the workspace; the store then has nerve's answer, his profile too.
   * Fails, changing nothing, when nerve refuses or the store has no membership of his.
   * @returns {Promise<WorkspaceMember>}
   */
  updateMember = (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate): Promise<WorkspaceMember> =>
    this.changes(async () => {
      const workspaceId = this.workspaceIdOf(workspaceSlug);
      const membership = await this.service.update(this.membership(workspaceId, userId).id, data);
      this.members.confirm(workspaceId, withMembership(membership));
      this.shareProfiles([membership]);
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
      const workspaceId = this.workspaceIdOf(workspaceSlug);
      await this.service.remove(this.membership(workspaceId, userId).id);
      this.members.confirm(workspaceId, ended(userId));
    });

  /** The membership of the member userId names in the workspace, as the store has it; fails when it has none. */
  private membership(workspaceId: string | undefined, userId: string): WorkspaceMember {
    const membership = this.members.get(workspaceId)?.[userId];
    if (!membership) throw new Error("Member not found");
    return membership;
  }

  /**
   * @description fetches a workspace's invitations, an admin's to fetch as nerve refuses anyone else, and shows them
   * with the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or
   * a change of session cut (Reconciled.fetch)
   * @returns {Promise<WorkspaceInvitation[] | undefined>}
   */
  fetchWorkspaceMemberInvitations = (
    workspace: Pick<Workspace, "id" | "slug">
  ): Promise<WorkspaceInvitation[] | undefined> =>
    this.invitations.fetch(workspace.id, () => this.invitationsService.list(workspace.slug));

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
      const workspaceId = this.workspaceIdOf(workspaceSlug);
      const created = await this.invitationsService.create(workspaceSlug, data);
      this.invitations.confirm(workspaceId, prepended(created));
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
      const workspaceId = this.workspaceIdOf(workspaceSlug);
      const changed = await this.invitationsService.update(invitationId, data);
      this.invitations.confirm(workspaceId, replaced(changed));
      return changed;
    });

  /**
   * @description deletes an invitation, which then leaves the list; fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteMemberInvitation = (workspaceSlug: string, invitationId: string): Promise<void> =>
    this.changes(async () => {
      const workspaceId = this.workspaceIdOf(workspaceSlug);
      await this.invitationsService.delete(invitationId);
      this.invitations.confirm(workspaceId, dropped(invitationId));
    });

  isUserSuspended = computedFn((userId: string, workspaceSlug: string | undefined) => {
    if (!workspaceSlug) return false;
    return this.getMemberships(workspaceSlug)?.[userId]?.is_active === false;
  });
}
