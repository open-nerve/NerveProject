/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { planningOn, type PlanningDomain } from "./planning-switch";

/** What a page of a project's cycles or modules reads of them. */
type PlanningReads = {
  /** A cycle or a module, its progress, its links, and the domain's archive (whether the domain is on or not). */
  readable: boolean;
  /** The domain's list of what is not archived, and what reads it: readable, while the domain is on. */
  listed: boolean;
};

/**
 * The gate of what a page of the address's project reads of its cycles or its modules (P6a spec 2.3, R3): a caller
 * who has a role in the project (a guest too: getProjectRoleByWorkspaceSlugAndProjectId, none for one who only sees
 * it, or for a project of another workspace than the address's) reads it while it is not archived; its list of the
 * domain, while the domain is turned on as well (planningOn). The address's workspace and project are the route's.
 * The wrapper of the project's pages decides alone that the caller is its member (nerve's read of it,
 * use-project-fetch.ts); the pages under it, and those beside it (the archives, the command palette), ask this.
 */
export function usePlanningReads(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  domain: PlanningDomain
): PlanningReads {
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const project = getProjectById(projectId);
  const role = workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  if (role === undefined || project === undefined || project.archived_at !== null)
    return { readable: false, listed: false };
  return { readable: true, listed: planningOn(project, domain) };
}
