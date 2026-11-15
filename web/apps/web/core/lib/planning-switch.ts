/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import type { IIssueDisplayProperties, TIssueGroupByOptions, TWorkItemFilterProperty } from "@nerve/types";

/** A domain of a project's planning: its cycles, its modules. */
export type PlanningDomain = "cycles" | "modules";

/** The field of a project that turns each domain on, as its features' page sets it. */
const SWITCH: Record<PlanningDomain, "cycle_view" | "module_view"> = { cycles: "cycle_view", modules: "module_view" };

/**
 * Whether the project's domain is turned on (Project.cycle_view, module_view): while it is off, the list of the
 * domain's page is its turned-off state and nothing of the list is read (P5a spec 2.2, P6a spec 2.2, 2.3). The one
 * predicate that the project's wrapper (use-project-fetch.ts), the gate of the domain's pages (use-planning-reads.ts)
 * and the permission store's write gates (planningWritesRefused) decide by; a store-free module of lib, for the wrapper
 * reads no permission store and a store depends on lib, not on hooks.
 */
export function planningOn(project: Pick<Project, "cycle_view" | "module_view">, domain: PlanningDomain): boolean {
  return project[SWITCH[domain]];
}

/** What the rules of a project's planning read of it: whether it is archived, and its switches. */
type PlanningProject = Pick<Project, "archived_at" | "cycle_view" | "module_view">;

/** nerve's problem code of a domain's thing the caller cannot see, as it says of a cycle or a module it does not give. */
const NOT_FOUND = { cycles: "planning.cycle_not_found", modules: "planning.module_not_found" } as const;

/** nerve's problem code of a domain turned off in the project (turnedOff, planning/app/lock.go). */
const DISABLED = { cycles: "planning.cycles_disabled", modules: "planning.modules_disabled" } as const;

/** Why a domain's list is not read: nerve's problem code of it (planningListRefusal). */
type PlanningListRefusal = (typeof NOT_FOUND)[PlanningDomain] | "project.archived" | (typeof DISABLED)[PlanningDomain];

/**
 * Why the caller's list of the project's domain is not read, by nerve's problem code (P7c spec 2.2, R2): he has no
 * role in the project, or it is not held (role is getProjectRoleByWorkspaceSlugAndProjectId's, by the address's
 * workspace and project, never the router's), the domain's own not-found code; the project is archived,
 * project.archived; the domain is turned off (planningOn), its disabled code; none while it is read, a guest's too.
 * The order is the read gate's (usePlanningReads), so that this is none exactly while its listed is true; the command
 * palette's page that opens a cycle or a module says it, by its message (problemMessageKey), while the gate is closed.
 */
export function planningListRefusal(
  project: PlanningProject | undefined,
  role: EUserPermissions | undefined,
  domain: PlanningDomain
): PlanningListRefusal | undefined {
  if (role === undefined || project === undefined) return NOT_FOUND[domain];
  if (project.archived_at !== null) return "project.archived";
  return planningOn(project, domain) ? undefined : DISABLED[domain];
}

/**
 * Whether the caller is offered the ways into the project's domain (P7c spec 2.2, R2): the command palette's commands
 * that go to its list or open one of it. His role in the project is an admin's or a member's (an admin's for the
 * workspace's admin, its guest too, as the permission store gives it) and its list is read: the project is not
 * archived and has the domain turned on (planningListRefusal). A guest reads the list by its address and has no way
 * in, as the sidebar declares it (its items' access: admins and members; shouldRender: the switch).
 */
export function planningEntryOn(
  project: PlanningProject | undefined,
  role: EUserPermissions | undefined,
  domain: PlanningDomain
): boolean {
  return (
    (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER) &&
    planningListRefusal(project, role, domain) === undefined
  );
}

/** A word of a work item list's options: a display property, a grouping, a filter. */
export type ListWord = keyof IIssueDisplayProperties | TIssueGroupByOptions | TWorkItemFilterProperty;

/**
 * The words of a list's options that name a project's planning, each its domain's (P7b spec 2.5, R8): the display
 * properties of a work item's cycle and modules, the groupings by its cycle and its module ("cycle" is both), the
 * filters by them.
 */
const PLANNING_WORDS: Partial<Record<NonNullable<ListWord>, PlanningDomain>> = {
  cycle: "cycles",
  modules: "modules",
  module: "modules",
  cycle_id: "cycles",
  module_id: "modules",
};

/**
 * Whether a list of a project's work items offers a word of its options (P7a spec 2.5, R11; P7b spec 2.5, R8): a word
 * of the planning only while the project is known and has the word's domain turned on; any other always. The display,
 * grouping and sub-grouping panels offer by it on a project's page (use-route-list-words.ts), and a project's table
 * shows its columns by it.
 */
export function listWordOn(project: Pick<Project, "cycle_view" | "module_view"> | undefined, word: ListWord): boolean {
  const domain = word === null ? undefined : PLANNING_WORDS[word];
  return !domain || (project !== undefined && planningOn(project, domain));
}
