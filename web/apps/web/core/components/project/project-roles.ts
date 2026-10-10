/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";

/**
 * A project's roles, the lowest first, as its role selects offer them, each labelled by ROLE: a select gives the
 * picked role's number, the ProjectRole nerve's ProjectMemberNew and ProjectMemberUpdate take (a string is refused).
 */
export const PROJECT_ROLES: ProjectRole[] = [5, 15, 20];

/** The caller as nerve decides his changes of a project's memberships: his role in its workspace, and in it. */
export type MembershipCaller = {
  workspaceRole: WorkspaceRole | undefined;
  projectRole: ProjectRole | null | undefined;
};

/**
 * A membership of the project as its members page shows it: whether it is the caller's own, its role, and its
 * member's role in the workspace.
 */
export type ShownMembership = { own: boolean; role: ProjectRole; workspaceRole: WorkspaceRole | undefined };

/** Whether role comes below the role than in PROJECT_ROLES' order, not by their numbers (as nerve's roleOrder). */
const isBelow = (role: ProjectRole, than: ProjectRole) => PROJECT_ROLES.indexOf(role) < PROJECT_ROLES.indexOf(than);

/**
 * The caller's role in the project when he changes its memberships at all, as nerve decides it (M3 design 3.5): its
 * admins, and its members who are the workspace's admins; undefined for anyone else.
 */
function managerRole({ workspaceRole, projectRole }: MembershipCaller): ProjectRole | undefined {
  if (projectRole === null || projectRole === undefined) return undefined;
  return projectRole === 20 || workspaceRole === 20 ? projectRole : undefined;
}

/**
 * The roles the caller may give a membership of the project, as nerve allows them (M3 design 3.5): none when he may
 * not change its role. One who is not the workspace's admin changes neither his own role nor one that is not below
 * his own, and gives no role that is not below his own; a workspace guest's role stays a guest's. The members page
 * offers these alone, so as not to offer what nerve refuses; hiding the others protects nothing (an admin may still
 * add a member as an admin, P5b spec §5).
 */
export function roleChoices(caller: MembershipCaller, membership: ShownMembership): ProjectRole[] {
  const callerRole = managerRole(caller);
  if (callerRole === undefined) return [];
  const assignable = membership.workspaceRole === 5 ? PROJECT_ROLES.filter((role) => role === 5) : PROJECT_ROLES;
  if (caller.workspaceRole === 20) return assignable;
  if (membership.own || !isBelow(membership.role, callerRole)) return [];
  return assignable.filter((role) => isBelow(role, callerRole));
}

/**
 * Whether the caller may remove a membership of the project, as nerve allows it (M3 design 3.5): not his own (he
 * leaves), nor one whose role is above his own role in the project, the workspace's admins neither.
 */
export function canRemove(caller: MembershipCaller, membership: ShownMembership): boolean {
  const callerRole = managerRole(caller);
  return callerRole !== undefined && !membership.own && !isBelow(callerRole, membership.role);
}
