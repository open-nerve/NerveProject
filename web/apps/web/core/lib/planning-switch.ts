/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
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
