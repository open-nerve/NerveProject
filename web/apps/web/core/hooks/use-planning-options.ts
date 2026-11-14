/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { useCycles } from "@/hooks/store/use-cycles";
import { useModules } from "@/hooks/store/use-modules";
import { cycleOptions, moduleOptions } from "@/lib/planning-directory";

/**
 * A work item's cycle options in its project (cycleOptions), and their read (P7a spec 2.3, R5): the project's cycles
 * that are not archived, read each time its dropdown opens (load), as the state dropdown's are. The read is the
 * project's and the caller's permissions': once either changes, its answer ends no loading and shows no refusal; the
 * store keeps an older fetch from writing over a newer one (Reconciled), each project's list apart.
 */
export function useCycleOptions(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { getProjectCycleIds, getCycleById, fetchCycles } = useCycles();
  const read = useDirectoryRead(workspaceSlug, projectId, projectId ? () => fetchCycles(projectId) : undefined);
  return { options: cycleOptions(getProjectCycleIds(projectId), getCycleById), read };
}

/** A work item's module options in its project (moduleOptions), and their read, as useCycleOptions. */
export function useModuleOptions(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { getProjectModuleIds, getModuleById, fetchModules } = useModules();
  const read = useDirectoryRead(workspaceSlug, projectId, projectId ? () => fetchModules(projectId) : undefined);
  return { options: moduleOptions(getProjectModuleIds(projectId), getModuleById), read };
}
