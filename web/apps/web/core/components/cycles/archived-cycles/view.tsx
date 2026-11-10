/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { CyclesList } from "@/components/cycles/list";
import { CyclesListMap } from "@/components/cycles/list/cycles-list-map";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
import { useCycles } from "@/hooks/store/use-cycles";
// local imports
import { filterCycleRows, NoMatchingCycles } from "../cycles-view";

export interface IArchivedCyclesView {
  workspaceSlug: string;
  projectId: string;
}

/**
 * A project's archived cycles in nerve's order, the latest archived first (R5), filtered and searched by the project's
 * own filters and search of its archive; each row opens the cycle's preview. Nothing matching, it says which to clear.
 */
export const ArchivedCyclesView = observer(function ArchivedCyclesView(props: IArchivedCyclesView) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const { getArchivedCycleIds, getCycleRow } = useCycles();
  const { getFilters, getSearchQuery } = useCycleFilters();
  // derived values
  const query = getSearchQuery(projectId, "archived");
  const rows = filterCycleRows(
    getArchivedCycleIds(projectId) ?? [],
    getCycleRow,
    getFilters(projectId, "archived"),
    query
  );

  if (rows.length === 0) return <NoMatchingCycles searched={query.trim() !== ""} />;

  return (
    <CyclesList workspaceSlug={workspaceSlug} projectId={projectId}>
      <CyclesListMap cycleIds={rows.map((row) => row.id)} projectId={projectId} workspaceSlug={workspaceSlug} />
    </CyclesList>
  );
});
