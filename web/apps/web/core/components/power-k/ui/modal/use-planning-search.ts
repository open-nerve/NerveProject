/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Cycle, Module, Project } from "@nerve/api-client";
// lib
import { planningEntryOn } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";
// store
import type { RootStore } from "@/store/root.store";

/** The cycles and the modules of the search's project whose names have its query, each with its project. */
type PlanningResults = {
  cycles: { cycle: Cycle; project: Project }[];
  modules: { module: Module; project: Project }[];
};

/**
 * The cycles and the modules the command palette's search names (P7c spec 2.4, R5): of the search's project alone,
 * projectId, none at the workspace's scope (its search is M7's). Of each domain while the caller is offered the ways
 * into it (planningEntryOn: his role an admin's or a member's, by the address's workspace and project, never the
 * router's; the project not archived; the domain on), of the store's list of the project's not archived, ended cycles
 * too (they are gone to, not joined), those whose names have needle, the lowercase query: filtered here, as the
 * search's projects are, for the palette filters nothing while it searches (wrapper.tsx). While the store holds no
 * such list, the open palette reads it by the key the project's pages read it by (its wrapper's, use-project-fetch.ts,
 * and the pages that open one, open-entity/), for /browse/:workItem has no wrapper: until nerve answers, or once it
 * refused, the domain names nothing, and holds up none of the search's other results.
 */
export function usePlanningSearch(
  rootStore: RootStore,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  needle: string,
  open: boolean
): PlanningResults {
  const { cycles, modules } = rootStore.planning;
  const project = rootStore.projectRoot.project.getProjectById(projectId);
  const role = workspaceSlug
    ? rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
    : undefined;
  // a domain with no way in names nothing, and reads nothing; one whose list the store does not hold, nothing yet
  const cycleIds = planningEntryOn(project, role, "cycles") ? cycles.getProjectCycleIds(projectId) : [];
  const moduleIds = planningEntryOn(project, role, "modules") ? modules.getProjectModuleIds(projectId) : [];
  useSessionSWR(open && !cycleIds ? ["PROJECT_CYCLES", projectId] : null, (id) => cycles.fetchCycles(id));
  useSessionSWR(open && !moduleIds ? ["PROJECT_MODULES", projectId] : null, (id) => modules.fetchModules(id));

  if (!project) return { cycles: [], modules: [] };
  const named = <T extends { name: string }>(item: T | undefined): item is T =>
    item !== undefined && item.name.toLowerCase().includes(needle);
  return {
    cycles: (cycleIds ?? [])
      .map((cycleId) => cycles.getCycleById(cycleId))
      .filter(named)
      .map((cycle) => ({ cycle, project })),
    modules: (moduleIds ?? [])
      .map((moduleId) => modules.getModuleById(moduleId))
      .filter(named)
      .map((module) => ({ module, project })),
  };
}
