/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useParams } from "react-router";
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";

/**
 * The route's cycle as its page's work items see it (P5b spec 2.6, R5). Read-only once the cycle has ended or is
 * archived, though nerve refuses only that work items join it, and while the page does not have it yet. Joinable by a
 * new work item while its project's cycles are on and it is not read-only. Whether the caller may write in the project
 * is the reader's to add.
 */
export function useCycleListState(): { readOnly: boolean; joinable: boolean } {
  const { projectId, cycleId } = useParams();
  const { getCycleById } = useCycles();
  const { getProjectById } = useProject();
  const cycle = getCycleById(cycleId);
  const readOnly = !cycle || cycle.status === "completed" || cycle.archived_at !== null;
  return { readOnly, joinable: !readOnly && getProjectById(projectId)?.cycle_view === true };
}
