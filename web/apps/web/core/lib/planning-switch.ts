/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
import type { IIssueDisplayProperties } from "@nerve/types";

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

/** The display properties of a project's planning, each its domain's: a work item's cycle, its modules. */
const DISPLAY_PROPERTIES: Partial<Record<keyof IIssueDisplayProperties, PlanningDomain>> = {
  cycle: "cycles",
  modules: "modules",
};

/**
 * Whether a display property shows for the work items of a project (P7a spec 2.5, R11): the cycle and the modules only
 * while the project is known and has their domain turned on; any other always. The display panel offers by it on a
 * project's page, and a project's table shows its columns by it.
 */
export function displayPropertyOn(
  project: Pick<Project, "cycle_view" | "module_view"> | undefined,
  property: keyof IIssueDisplayProperties
): boolean {
  const domain = DISPLAY_PROPERTIES[property];
  return !domain || (project !== undefined && planningOn(project, domain));
}
