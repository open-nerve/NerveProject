/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import type { IWorkItemFilterInstance, TWorkItemFilterCondition } from "@nerve/shared-state";
import type { TWorkItemFilterProperty } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { toFilterArray } from "@nerve/utils";
// components
import type { RowChoice } from "@/components/core/sidebar/distribution-stats";
import { createFilterUpdateHandler } from "@/components/core/sidebar/progress-stats/shared";
import { ProgressTabs } from "@/components/core/sidebar/progress-tabs";
// hooks
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";

type Props = {
  cycleId: string;
  projectId: string;
  /** The progress the sidebar shows (cycleCounts' choice): its counts, its assignees and its labels. */
  progress: Progress;
};

/** The values of property the filters' condition `in` holds (findFirstConditionByPropertyAndOperator's, observed). */
const selectedOf = (filter: IWorkItemFilterInstance, property: TWorkItemFilterProperty) =>
  toFilterArray(
    filter.allConditions.find((condition) => condition.property === property && condition.operator === "in")?.value
  ).map(String);

/**
 * The breakdowns of a cycle's progress (P5a spec 2.4, 2.5), each a tab (ProgressTabs), the tab chosen kept for the
 * cycle (localStorage). While the cycle's page shows its work items by its filters (PlanningPageLayoutRoot's instance
 * of them), a row of a state group, an assignee or a label toggles its value in them, saved as the page's own change of
 * them, and says whether it is chosen (P5b spec 2.8, R8); the rows are plain before the page has its filters, and in
 * the list page's preview (?peekCycle), which never has them: only the cycle's page makes them.
 */
export const CycleProgressStats = observer(function CycleProgressStats(props: Props) {
  const { cycleId, projectId, progress } = props;
  // hooks
  const { getFilter, updateFilterValueFromSidebar } = useWorkItemFilters();
  const filter = getFilter(EIssuesStoreType.CYCLE, cycleId);
  const toggle = (property: TWorkItemFilterProperty): RowChoice => {
    if (!filter) return {};
    const values = selectedOf(filter, property);
    const update = (condition: TWorkItemFilterCondition) =>
      updateFilterValueFromSidebar(EIssuesStoreType.CYCLE, cycleId, condition);
    return { selected: values, choose: createFilterUpdateHandler(property, values, update) };
  };

  return (
    <ProgressTabs
      progress={progress}
      projectId={projectId}
      storageKey={`cycle-progress-tab-${cycleId}`}
      choices={{ states: toggle("state_group"), assignees: toggle("assignee_id"), labels: toggle("label_id") }}
    />
  );
});
