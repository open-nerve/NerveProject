/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { CycleListItem } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import type { CycleFilters } from "@nerve/utils";
import { matchesCycleFilters, orderCycleRows } from "@nerve/utils";
// assets
import AllFiltersImage from "@/app/assets/empty-state/cycle/all-filters.svg?url";
import NameFilterImage from "@/app/assets/empty-state/cycle/name-filter.svg?url";
// components
import { ActiveCycleRoot } from "@/components/cycles/active-cycle/root";
import { CycleListSection, CyclesList } from "@/components/cycles/list";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
import { useCycles } from "@/hooks/store/use-cycles";

/**
 * The rows of cycleIds the store gives that pass filters and whose names hold query, whatever its case, in the order
 * of cycleIds.
 */
export function filterCycleRows(
  cycleIds: string[],
  rowOf: (cycleId: string) => CycleListItem | undefined,
  filters: CycleFilters,
  query: string
): CycleListItem[] {
  const searched = query.trim().toLowerCase();
  return cycleIds.flatMap((cycleId) => {
    const row = rowOf(cycleId);
    return row && matchesCycleFilters(row, filters) && row.name.toLowerCase().includes(searched) ? [row] : [];
  });
}

/** That none of a project's cycles match its filters and its search, and which to clear to see them all. */
export function NoMatchingCycles({ searched }: { searched: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="grid h-full w-full place-items-center">
      <div className="text-center">
        <img
          src={searched ? NameFilterImage : AllFiltersImage}
          className="mx-auto h-36 w-36 object-contain sm:h-48 sm:w-48"
          alt=""
        />
        <h5 className="mt-7 mb-1 text-18 font-medium">{t("project_cycles.no_matching_cycles")}</h5>
        <p className="text-14 text-placeholder">
          {searched
            ? t("project_cycles.remove_search_criteria_to_see_all_cycles")
            : t("project_cycles.remove_filters_to_see_all_cycles")}
        </p>
      </div>
    </div>
  );
}

export interface ICyclesView {
  workspaceSlug: string;
  projectId: string;
}

/**
 * A project's cycles list (P5a spec 2.5, R5), of its cycles not archived as nerve lists them, filtered and searched by
 * the project's own filters and search: the cycle in progress (its card), then "upcoming", open, the cycles not yet
 * ended in the list's order (orderCycleRows: the drafts too, after the upcoming ones), then "completed", closed, the
 * ended ones in nerve's order. Nothing matching, it says which to clear.
 */
export const CyclesView = observer(function CyclesView(props: ICyclesView) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectCycleIds, getCycleRow, getActiveCycleId } = useCycles();
  const { getFilters, getSearchQuery } = useCycleFilters();
  // derived values
  const query = getSearchQuery(projectId, "default");
  const rows = filterCycleRows(
    getProjectCycleIds(projectId) ?? [],
    getCycleRow,
    getFilters(projectId, "default"),
    query
  );

  if (rows.length === 0) return <NoMatchingCycles searched={query.trim() !== ""} />;

  const activeCycleId = getActiveCycleId(projectId);
  const upcoming = orderCycleRows(rows).filter((row) => row.id !== activeCycleId);
  const completed = rows.filter((row) => row.status === "completed");

  return (
    <CyclesList workspaceSlug={workspaceSlug} projectId={projectId}>
      <ActiveCycleRoot workspaceSlug={workspaceSlug} projectId={projectId} />
      <CycleListSection
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        status="upcoming"
        title={t("project_cycles.upcoming_cycle.label")}
        cycleIds={upcoming.map((row) => row.id)}
        defaultOpen
      />
      <CycleListSection
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        status="completed"
        title={t("project_cycles.completed_cycle.label")}
        cycleIds={completed.map((row) => row.id)}
      />
    </CyclesList>
  );
});
