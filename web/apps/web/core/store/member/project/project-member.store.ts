/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { omit, sortBy, union, without } from "lodash-es";
import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import type { ApiClient, MemberUser, Project, ProjectMember, ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ProjectMembersService } from "@/services/project/project-members.service";
// store
import type { IProjectStore } from "@/store/project/project.store";
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";
import type { IUserStore } from "@/store/user";
// local imports
import type { IMemberRootStore } from "../index";
import { sortProjectMembers } from "../utils";
import type { IProjectMemberFiltersStore } from "./project-member-filters.store";
import { ProjectMemberFiltersStore } from "./project-member-filters.store";

/** A project's active memberships by the member's account id. */
type Memberships = Record<string, ProjectMember>;

/** A membership of a project with its member's profile, which the workspace's members give (M3 design 7.2). */
export type IProjectMemberDetails = ProjectMember & { member: MemberUser };

export interface IProjectMemberStore {
  // filters store
  filters: IProjectMemberFiltersStore;
  // computed
  projectMemberIds: string[] | null;
  // computed actions
  getProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
  getProjectMemberIds: (projectId: string, includeGuestUsers: boolean) => string[] | null;
  getFilteredProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
  // fetch actions
  fetchProjectMembers: (projectId: string) => Promise<Memberships | undefined>;
  // changes
  bulkAddMembersToProject: (projectId: string, data: ProjectMembersAdd) => Promise<ProjectMember[]>;
  updateMemberRole: (projectId: string, userId: string, role: ProjectRole) => Promise<ProjectMember>;
  removeMemberFromProject: (projectId: string, userId: string) => Promise<void>;
}

/** The memberships by their members' account ids. */
const byMember = (memberships: ProjectMember[]): Memberships =>
  Object.fromEntries(memberships.map((membership) => [membership.member_id, membership]));

/**
 * The members of the projects of a session (M3 design 7.3), each project's by its id: its service sends with the
 * session's client, which the RootStore of the session hands down. Changes go one at a time and the store writes
 * nerve's answers (v0 design 7.7); fetches do not queue. The members of a project the project store no longer gives
 * (deleted, left, or of a workspace no longer the caller's) do not show. What a change makes of the project, its
 * member_ids and the caller's own role (member_role), the project store shows too.
 */
export class ProjectMemberStore implements IProjectMemberStore {
  /** Each project's memberships, reconciled between their fetches and the changes nerve confirmed (reconciled.ts). */
  private readonly members = new ReconciledByKey<Memberships>();
  // filters store
  filters: IProjectMemberFiltersStore;
  // stores
  private readonly routerStore: IRouterStore;
  private readonly userStore: IUserStore;
  /** The users the stores read, which the workspace's members' profiles fill. */
  private readonly memberRoot: Pick<IMemberRootStore, "memberMap">;
  /** The projects the caller sees, which show what the changes of their members make of them. */
  private readonly projects: Pick<IProjectStore, "getProjectById" | "confirmProject">;
  // services
  private readonly service: ProjectMembersService;
  /** The changes of the memberships, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(_memberRoot: Pick<IMemberRootStore, "memberMap">, _rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      projectMemberIds: computed,
      // actions
      fetchProjectMembers: action,
      bulkAddMembersToProject: action,
      updateMemberRole: action,
      removeMemberFromProject: action,
    });
    this.routerStore = _rootStore.router;
    this.userStore = _rootStore.user;
    this.memberRoot = _memberRoot;
    this.projects = _rootStore.projectRoot.project;
    this.filters = new ProjectMemberFiltersStore();
    this.service = new ProjectMembersService(api);
  }

  /** The project's memberships, once fetched, while the project store gives the project. */
  private membershipsOf(projectId: string): Memberships | undefined {
    return this.projects.getProjectById(projectId) ? this.members.get(projectId) : undefined;
  }

  /**
   * @description the members of the address's project, by the members page's filters and order; null until fetched
   * or while it has none
   */
  get projectMemberIds() {
    const projectId = this.routerStore.projectId;
    if (!projectId) return null;
    const members = Object.values(this.membershipsOf(projectId) ?? {});
    if (members.length === 0) return null;
    const sortedMembers = sortProjectMembers(
      members,
      this.memberRoot.memberMap,
      (member) => member.member_id,
      this.filters.filtersMap[projectId]
    );
    return sortedMembers.map((member) => member.member_id);
  }

  /** @description a member's membership of the project with his profile; null without either */
  getProjectMemberDetails = computedFn((userId: string, projectId: string): IProjectMemberDetails | null => {
    const membership = this.membershipsOf(projectId)?.[userId];
    const member = this.memberRoot.memberMap[userId];
    return membership && member ? { ...membership, member } : null;
  });

  /** @description the project's members, the caller first, then by display name; null until fetched */
  getProjectMemberIds = computedFn((projectId: string, includeGuestUsers: boolean): string[] | null => {
    const memberships = this.membershipsOf(projectId);
    if (!memberships) return null;
    const members = Object.values(memberships).filter(
      (membership) => includeGuestUsers || membership.role !== EUserPermissions.GUEST
    );
    return sortBy(members, [
      (membership) => membership.member_id !== this.userStore.data?.id,
      (membership) => this.memberRoot.memberMap[membership.member_id]?.display_name.toLowerCase(),
    ]).map((membership) => membership.member_id);
  });

  /** @description as getProjectMemberDetails, for a member the members page's filters let through */
  getFilteredProjectMemberDetails = computedFn((userId: string, projectId: string): IProjectMemberDetails | null => {
    const memberships = Object.values(this.membershipsOf(projectId) ?? {});
    const shown = this.filters.getFilteredMemberIds(
      memberships,
      this.memberRoot.memberMap,
      (membership) => membership.member_id,
      projectId
    );
    return shown.includes(userId) ? this.getProjectMemberDetails(userId, projectId) : null;
  });

  /**
   * @description fetches a project's members, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectMembers = (projectId: string): Promise<Memberships | undefined> =>
    this.members.fetch(projectId, async () => byMember(await this.service.list(projectId)));

  /**
   * @description adds members of the workspace to a project, all of them or none; the store then has their
   * memberships as nerve answers them. Fails, changing nothing, when nerve refuses.
   */
  bulkAddMembersToProject = (projectId: string, data: ProjectMembersAdd): Promise<ProjectMember[]> =>
    this.changes(async () => {
      const added = await this.service.add(projectId, data);
      this.members.confirm(projectId, (memberships) => ({ ...memberships, ...byMember(added) }));
      const ids = added.map((membership) => membership.member_id);
      this.confirmOnProject(projectId, (project) => ({ ...project, member_ids: union(project.member_ids, ids) }));
      return added;
    });

  /**
   * @description changes a member's role; the store then has nerve's answer, and the project the caller's own role
   * when it is his. Fails, changing nothing, when nerve refuses or the store has no membership of his.
   */
  updateMemberRole = (projectId: string, userId: string, role: ProjectRole): Promise<ProjectMember> =>
    this.changes(async () => {
      const membership = await this.service.update(this.membership(projectId, userId).id, { role });
      this.members.confirm(projectId, (memberships) => ({ ...memberships, [userId]: membership }));
      if (userId === this.userStore.data?.id) {
        this.confirmOnProject(projectId, (project) => ({ ...project, member_role: membership.role }));
      }
      return membership;
    });

  /**
   * @description ends a member's membership of a project, which the store then no longer has. Fails, changing
   * nothing, when nerve refuses (the caller's own: he leaves the project instead) or the store has no membership of
   * his.
   */
  removeMemberFromProject = (projectId: string, userId: string): Promise<void> =>
    this.changes(async () => {
      await this.service.remove(this.membership(projectId, userId).id);
      this.members.confirm(projectId, (memberships) => omit(memberships, userId));
      this.confirmOnProject(projectId, (project) => ({ ...project, member_ids: without(project.member_ids, userId) }));
    });

  /** The membership of the member userId names in the project, as the store has it; fails when it has none. */
  private membership(projectId: string, userId: string): ProjectMember {
    const membership = this.membershipsOf(projectId)?.[userId];
    if (!membership) throw new Error("Member not found");
    return membership;
  }

  /** What a change nerve confirmed makes of the project, wherever the project store shows it. */
  private confirmOnProject(projectId: string, change: Change<Project>): void {
    const project = this.projects.getProjectById(projectId);
    if (project) this.projects.confirmProject(project, change);
  }
}
