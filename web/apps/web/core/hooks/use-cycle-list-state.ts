/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useParams } from "react-router";
import type { Cycle, Project } from "@nerve/api-client";
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";

/** What a cycle's page's work items can be: read-only, or joinable by a new work item. */
export type CycleListState = { readOnly: boolean; joinable: boolean };

/**
 * A cycle's page's work items as the page shows its cycle and project (P5b spec 2.5, 2.6, R5). Read-only once the
 * cycle has ended or is archived, though nerve refuses only that work items join it, and while the page does not have
 * it yet. Joinable by a new work item while its project's cycles are on and it is not read-only. Whether the caller may
 * write in the project is the reader's to add.
 */
export function cycleListState(cycle: Cycle | undefined, project: Project | undefined): CycleListState {
  const readOnly = !cycle || cycle.status === "completed" || cycle.archived_at !== null;
  return { readOnly, joinable: !readOnly && project?.cycle_view === true };
}

/** The route's cycle's list state (cycleListState), as its page shows it. */
export function useCycleListState(): CycleListState {
  const { projectId, cycleId } = useParams();
  const { getCycleById } = useCycles();
  const { getProjectById } = useProject();
  return cycleListState(getCycleById(cycleId), getProjectById(projectId));
}
