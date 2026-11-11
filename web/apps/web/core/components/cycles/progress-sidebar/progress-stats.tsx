/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Tab } from "@headlessui/react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { STATE_GROUPS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { IWorkItemFilterInstance, TWorkItemFilterCondition } from "@nerve/shared-state";
import type { TWorkItemFilterProperty } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { cn, toFilterArray } from "@nerve/utils";
// assets
import emptyLabel from "@/app/assets/empty-state/empty_label.svg?url";
import emptyMembers from "@/app/assets/empty-state/empty_members.svg?url";
// components
import { createFilterUpdateHandler, PROGRESS_STATS } from "@/components/core/sidebar/progress-stats/shared";
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
// hooks
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import useLocalStorage from "@/hooks/use-local-storage";
// local imports
import { AssigneeStats, LabelStats, useDistributionRows } from "../distribution-stats";

/** The state groups whose work items a cycle's progress counts, in their order; the cancelled ones it does not. */
const COUNTED = ["backlog", "unstarted", "started", "completed"] as const;

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
 * The breakdowns of a cycle's progress (P5a spec 2.4, 2.5), each a tab: by state, each group's share of the work items
 * not cancelled (progressPercent's, the cancelled said apart); by assignee and by label, as nerve counts them, named by
 * the project's members and labels (distribution-rows.ts). The tab chosen is kept for the cycle (localStorage).
 * While the cycle's page shows its work items by its filters (CycleLayoutRoot's instance of them), a row of a state
 * group, an assignee or a label toggles its value in them, saved as the page's own change of them, and says whether it
 * is chosen (P5b spec 2.8, R8); the rows are plain before the page has its filters, and in the list page's preview
 * (?peekCycle), which never has them: only the cycle's page makes them.
 */
export const CycleProgressStats = observer(function CycleProgressStats(props: Props) {
  const { cycleId, projectId, progress } = props;
  // hooks
  const { t } = useTranslation();
  const { assignees, labels } = useDistributionRows(progress, projectId);
  const { getFilter, updateFilterValueFromSidebar } = useWorkItemFilters();
  const filter = getFilter(EIssuesStoreType.CYCLE, cycleId);
  const toggle = (property: TWorkItemFilterProperty): { choose?: (value: string) => void; selected?: string[] } => {
    if (!filter) return {};
    const values = selectedOf(filter, property);
    const update = (condition: TWorkItemFilterCondition) =>
      updateFilterValueFromSidebar(EIssuesStoreType.CYCLE, cycleId, condition);
    return { selected: values, choose: createFilterUpdateHandler(property, values, update) };
  };
  const { choose: chooseState, selected: statesChosen } = toggle("state_group");
  const { storedValue: currentTab, setValue: setCycleTab } = useLocalStorage(
    `cycle-progress-tab-${cycleId}`,
    "stat-assignees"
  );
  // derived values
  const selected = Math.max(
    0,
    PROGRESS_STATS.findIndex((stat) => stat.key === currentTab)
  );
  const counted = progress.total_issues - progress.cancelled_issues;

  return (
    <Tab.Group
      as="div"
      selectedIndex={selected}
      onChange={(index: number) => setCycleTab(PROGRESS_STATS[index]?.key ?? "stat-assignees")}
    >
      <Tab.List as="div" className="flex w-full items-center justify-between gap-2 rounded-md bg-layer-2 p-1 text-11">
        {PROGRESS_STATS.map((stat, index) => (
          <Tab
            className={cn(
              "w-full cursor-pointer rounded-sm p-1 text-primary transition-all outline-none focus:outline-none",
              index === selected
                ? "bg-layer-transparent-active text-secondary"
                : "text-placeholder hover:text-secondary"
            )}
            key={stat.key}
          >
            {t(stat.i18n_title)}
          </Tab>
        ))}
      </Tab.List>
      <Tab.Panels className="py-3 text-secondary">
        <Tab.Panel>
          <ul>
            {COUNTED.map((group) => (
              <li key={group}>
                <SingleProgressStats
                  title={<span className="text-11">{t(STATE_GROUPS[group].i18n_label)}</span>}
                  completed={progress[`${group}_issues` as const]}
                  total={counted}
                  {...(chooseState && { onClick: () => chooseState(group), selected: statesChosen?.includes(group) })}
                />
              </li>
            ))}
          </ul>
          {progress.cancelled_issues > 0 && (
            <p className="p-1 text-11 text-tertiary">
              {t("project_cycles.active_cycle.cancelled_excluded", { count: progress.cancelled_issues })}
            </p>
          )}
        </Tab.Panel>
        <Tab.Panel>
          {assignees.length > 0 ? (
            <AssigneeStats rows={assignees} {...toggle("assignee_id")} />
          ) : (
            <NoRows image={emptyMembers}>{t("no_assignee")}</NoRows>
          )}
        </Tab.Panel>
        <Tab.Panel>
          {labels.length > 0 ? (
            <LabelStats rows={labels} {...toggle("label_id")} />
          ) : (
            <NoRows image={emptyLabel}>{t("no_labels_yet")}</NoRows>
          )}
        </Tab.Panel>
      </Tab.Panels>
    </Tab.Group>
  );
});

/** A breakdown without a row: no work item of the cycle is counted in it. */
function NoRows({ image, children }: { image: string; children: ReactNode }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <div className="flex h-20 w-20 items-center justify-center rounded-full bg-layer-1">
        <img src={image} className="h-12 w-12 object-contain" alt="" aria-hidden="true" />
      </div>
      <p className="text-14 text-tertiary">{children}</p>
    </div>
  );
}
