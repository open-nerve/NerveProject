/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCycles } from "@/hooks/store/use-cycles";
import { useModules } from "@/hooks/store/use-modules";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import type { PlanningDomain } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";
import type { IPlanningStore } from "@/store/planning";

/** A project's list of a domain: the key every page reads it by, the ids the store holds of it, and its fetch. */
type List = {
  key: string;
  held: (projectId: string | undefined) => string[] | undefined;
  fetch: (projectId: string) => Promise<unknown>;
};

/** What names a domain's values in the planning store: whether it names one, and the two lists of a project. */
type Directory = { names: (id: string) => boolean; list: List; archive: List };

/**
 * Each domain's directory: the project's list of what is not archived, by the key its wrapper reads it by
 * (use-project-fetch.ts), and its archive, by the key its archive's page reads it by (archived-cycles/root.tsx,
 * archived-modules/root.tsx), so that a list read for a name is the list those pages read, and theirs is its.
 */
const DIRECTORIES: Record<PlanningDomain, (planning: Pick<IPlanningStore, "cycles" | "modules">) => Directory> = {
  cycles: ({ cycles }) => ({
    names: (id) => cycles.getCycleById(id) !== undefined,
    list: { key: "PROJECT_CYCLES", held: cycles.getProjectCycleIds, fetch: cycles.fetchCycles },
    archive: { key: "PROJECT_ARCHIVED_CYCLES", held: cycles.getArchivedCycleIds, fetch: cycles.fetchArchivedCycles },
  }),
  modules: ({ modules }) => ({
    names: (id) => modules.getModuleById(id) !== undefined,
    list: { key: "PROJECT_MODULES", held: modules.getProjectModuleIds, fetch: modules.fetchModules },
    archive: {
      key: "PROJECT_ARCHIVED_MODULES",
      held: modules.getArchivedModuleIds,
      fetch: modules.fetchArchivedModules,
    },
  }),
};

/**
 * Reads, on demand, what names a work item's values of a domain in its project (P7a spec 2.3, R14): its cycle or its
 * modules, which show by their names, ended and archived ones too, as the planning store has them (getCycleById,
 * getModuleById). While the store cannot name one of them, and the caller reads the project's domain (a role in the
 * project, which is not archived, and the domain turned on: usePlanningReads), it reads the project's list of what is
 * not archived while the store has none, then the project's archive while it has none; one read a project, whoever
 * asks, by the keys the project's pages share (DIRECTORIES), never one a value. A value still unnamed once the store
 * has both lists (deleted meanwhile) shows no name. Its caller observes the store, so that it reads on as the store
 * answers.
 */
export function usePlanningNames(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  domain: PlanningDomain,
  valueIds: readonly string[]
): void {
  const { listed } = usePlanningReads(workspaceSlug, projectId, domain);
  const { names, list, archive } = DIRECTORIES[domain]({ cycles: useCycles(), modules: useModules() });
  const unnamed = listed && valueIds.some((id) => !names(id));
  const readList = unnamed && list.held(projectId) === undefined;
  const readArchive = unnamed && !readList && archive.held(projectId) === undefined;
  useSessionSWR(readList ? [list.key, projectId] : null, (id) => list.fetch(id));
  useSessionSWR(readArchive ? [archive.key, projectId] : null, (id) => archive.fetch(id));
}

/**
 * The project's list of what is not archived of a domain, read while the caller reads the project's domain
 * (usePlanningReads), by the key the project's pages read it by (DIRECTORIES): whether it has answered, that is
 * whether the store holds it, its read failed, or nothing reads it (the domain off, no role).
 */
function useListAnswered(workspaceSlug: string | undefined, projectId: string | undefined, domain: PlanningDomain) {
  const { listed } = usePlanningReads(workspaceSlug, projectId, domain);
  const { list } = DIRECTORIES[domain]({ cycles: useCycles(), modules: useModules() });
  const read = useSessionSWR(listed ? [list.key, projectId] : null, (id) => list.fetch(id));
  return !listed || list.held(projectId) !== undefined || read.error !== undefined;
}

/**
 * Reads a project's two lists of what is not archived, its cycles and its modules, of which a work item's options are
 * (P7a spec 2.6, re-72 F7): a new work item's form reads them as it opens with its project, by the keys its pages share
 * (one read, whoever asks), and holds its values, its fields loading, until both have answered (useListAnswered).
 * Whether both have. Its caller observes the store.
 */
export function usePlanningListsAnswered(workspaceSlug: string | undefined, projectId: string | undefined): boolean {
  const cycles = useListAnswered(workspaceSlug, projectId, "cycles");
  const modules = useListAnswered(workspaceSlug, projectId, "modules");
  return cycles && modules;
}
