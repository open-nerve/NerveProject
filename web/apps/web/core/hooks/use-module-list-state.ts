/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Module, Project } from "@nerve/api-client";

/** What a module's page's work items can be: read-only, or joinable by a new work item. */
type ModuleListState = { readOnly: boolean; joinable: boolean };

/**
 * A module's page's work items as the page shows its module and project (P6b spec 2.7, R9). Read-only once the module
 * is archived, though nerve refuses only that work items join it, and while the page does not have it yet; its status
 * (completed, cancelled or any other) is no matter, a module taking work items in each (M6 design 3.3). Joinable by a
 * new work item while its project's modules are on and it is not read-only. Whether the caller may write in the
 * project is the reader's to add. The cycle's page has rules of its own (use-cycle-list-state.ts, P6b spec 2.2).
 */
export function moduleListState(module: Module | undefined, project: Project | undefined): ModuleListState {
  const readOnly = !module || module.archived_at !== null;
  return { readOnly, joinable: !readOnly && project?.module_view === true };
}
