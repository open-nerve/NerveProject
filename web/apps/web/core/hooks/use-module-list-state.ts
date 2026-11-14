/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useParams } from "react-router";
import type { Module, Project } from "@nerve/api-client";
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";

/** What a module's page's work items can be: read-only, or joinable by a new work item. */
type ModuleListState = { readOnly: boolean; joinable: boolean };

/**
 * A module's page's work items as the page shows its module and project (P6b spec 2.7, R9). Read-only once the module
 * is archived, though nerve refuses only that work items join it, and while the page does not have it: not yet, or the
 * module held is another project's than the page's, as the page's body and crumb say (spec 5, review re-71 M2); its
 * status (completed, cancelled or any other) is no matter, a module taking work items in each (M6 design 3.3). Joinable
 * by a new work item while its project's modules are on and it is not read-only. Whether the caller may write in the
 * project is the reader's to add. The cycle's page has rules of its own (use-cycle-list-state.ts, P6b spec 2.2).
 */
export function moduleListState(module: Module | undefined, project: Project | undefined): ModuleListState {
  const own = project === undefined || module?.project_id === project.id ? module : undefined;
  const readOnly = !own || own.archived_at !== null;
  return { readOnly, joinable: !readOnly && project?.module_view === true };
}

/** The route's module's list state (moduleListState), as its page shows it. */
export function useModuleListState(): ModuleListState {
  const { projectId, moduleId } = useParams();
  const { getModuleById } = useModules();
  const { getProjectById } = useProject();
  return moduleListState(getModuleById(moduleId), getProjectById(projectId));
}
