/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
import { Tab } from "@headlessui/react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// assets
import darkAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-dark.webp?url";
import lightAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-light.webp?url";
import darkLabelAsset from "@/app/assets/empty-state/active-cycle/label-dark.webp?url";
import lightLabelAsset from "@/app/assets/empty-state/active-cycle/label-light.webp?url";
// components
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
// hooks
import useLocalStorage from "@/hooks/use-local-storage";
// local imports
import { AssigneeStats, LabelStats, useDistributionRows } from "../distribution-stats";
import { ActiveCyclePriorityIssues } from "./priority-issues";
import type { EnterFiltered } from "./use-enter-filtered";

/** The card's tabs, in their order (P5b spec 2.7), each as the browser keeps it (localStorage activeCycleTab). */
const TABS = ["Priority-Issues", "Assignees", "Labels"] as const;
/** The priorities of the tab of priority work items, which its row sets the cycle's page to (P5b spec 2.7). */
const URGENT_AND_HIGH = ["urgent", "high"];

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The cycle in progress the card shows. */
  cycleId: string;
  /** The progress the card shows (cycleCounts' choice): its assignees and its labels. */
  progress: Progress;
  /** Takes the cycle's page to the work items of a condition (useEnterFiltered). */
  enterFiltered: EnterFiltered;
};

/**
 * The work items of the cycle in progress (P5a spec 2.4, 2.5, R10; P5b spec 2.7), each a tab, in the spec's order: its
 * urgent and high ones (ActiveCyclePriorityIssues), then its breakdowns by assignee (the one first shown) and by label,
 * as nerve counts them, named by the project's members and labels (distribution-stats.tsx); without a row, how to see
 * one. The tab chosen is kept (localStorage). A row of a work item, an assignee or a label (not the rows of none) takes
 * the cycle's page to its work items: the urgent and high ones, previewing the row's there, or the assignee's, or the
 * label's (P5b spec 2.7, 2.8).
 */
export const ActiveCycleStats = observer(function ActiveCycleStats(props: Props) {
  const { workspaceSlug, projectId, cycleId, progress, enterFiltered } = props;
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  const { assignees, labels } = useDistributionRows(progress, projectId);
  // local storage
  const { storedValue: tab, setValue: setTab } = useLocalStorage<(typeof TABS)[number]>("activeCycleTab", "Assignees");
  // derived values
  const light = resolvedTheme === "light";

  return (
    <div className="col-span-1 flex min-h-[17rem] flex-col gap-4 overflow-hidden rounded-lg border border-subtle bg-surface-1 p-4 lg:col-span-2 xl:col-span-1">
      <Tab.Group
        selectedIndex={Math.max(0, TABS.indexOf(tab ?? "Assignees"))}
        onChange={(index: number) => setTab(TABS[index] ?? "Assignees")}
      >
        <Tab.List
          as="div"
          className="relative grid grid-cols-3 rounded-sm border-[0.5px] border-subtle bg-layer-1 p-[1px]"
        >
          {[
            t("project_cycles.active_cycle.priority_issue"),
            t("project_cycles.active_cycle.assignees"),
            t("project_cycles.active_cycle.labels"),
          ].map((name) => (
            <Tab
              key={name}
              className={({ selected }) =>
                cn(
                  "relative z-[1] rounded-[3px] py-1.5 text-11 font-semibold text-placeholder transition duration-500 focus:outline-none",
                  { "bg-surface-1 text-tertiary": selected, "hover:text-tertiary": !selected }
                )
              }
            >
              {name}
            </Tab>
          ))}
        </Tab.List>
        <Tab.Panels>
          <StatsPanel>
            <ActiveCyclePriorityIssues
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              cycleId={cycleId}
              openIssue={(issueId) =>
                enterFiltered({ property: "priority", operator: "in", value: URGENT_AND_HIGH }, issueId)
              }
            />
          </StatsPanel>
          <StatsPanel>
            {assignees.length > 0 ? (
              <AssigneeStats
                rows={assignees}
                choose={(id) => enterFiltered({ property: "assignee_id", operator: "in", value: [id] })}
              />
            ) : (
              <NoRows
                title={t("active_cycle.empty_state.assignee.title")}
                asset={light ? lightAssigneeAsset : darkAssigneeAsset}
              />
            )}
          </StatsPanel>
          <StatsPanel>
            {labels.length > 0 ? (
              <LabelStats
                rows={labels}
                choose={(id) => enterFiltered({ property: "label_id", operator: "in", value: [id] })}
              />
            ) : (
              <NoRows
                title={t("active_cycle.empty_state.label.title")}
                asset={light ? lightLabelAsset : darkLabelAsset}
              />
            )}
          </StatsPanel>
        </Tab.Panels>
      </Tab.Group>
    </div>
  );
});

/** A tab's rows, scrolled within the card. */
function StatsPanel({ children }: { children: ReactNode }) {
  return (
    <Tab.Panel
      as="div"
      className="vertical-scrollbar scrollbar-sm flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
    >
      {children}
    </Tab.Panel>
  );
}

/** A breakdown without a row: how to see one. */
function NoRows({ title, asset }: { title: string; asset: string }) {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <SimpleEmptyState title={title} assetPath={asset} />
    </div>
  );
}
