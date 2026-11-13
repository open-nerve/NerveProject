/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Tab } from "@headlessui/react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { STATE_GROUPS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// assets
import emptyLabel from "@/app/assets/empty-state/empty_label.svg?url";
import emptyMembers from "@/app/assets/empty-state/empty_members.svg?url";
// components
import { PROGRESS_STATS } from "@/components/core/sidebar/progress-stats/shared";
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
// hooks
import useLocalStorage from "@/hooks/use-local-storage";
// local imports
import type { RowChoice } from "./distribution-stats";
import { AssigneeStats, choiceOf, LabelStats, useDistributionRows } from "./distribution-stats";

/** The state groups whose work items a progress counts, in their order; the cancelled ones it does not. */
const COUNTED = ["backlog", "unstarted", "started", "completed"] as const;

/** A tab, by its key: the breakdown by state, by assignee, by label. */
type ProgressTab = (typeof PROGRESS_STATS)[number]["key"];

type Props = {
  /** The progress the tabs break down: its counts, its assignees and its labels. */
  progress: Progress;
  /** The project whose members and labels name the rows. */
  projectId: string;
  /** The key of localStorage the tab chosen is kept under: the cycle's, the module's. */
  storageKey: string;
  /**
   * What choosing a row of a state group, of an assignee and of a label does, on a page whose filters the rows
   * toggle; without it, the rows are plain.
   */
  choices?: { states: RowChoice; assignees: RowChoice; labels: RowChoice };
};

/**
 * The breakdowns of a cycle's or a module's progress, each a tab (P5a spec 2.4, 2.5; P6a spec 2.5, R8): by state, each
 * group's share of the work items not cancelled (progressPercent's, the cancelled said apart); by assignee and by
 * label, as nerve counts them, named by the project's members and labels (distribution-rows.ts). Each tab shows its
 * own breakdown: the panels are laid out from the tabs' own list, so they cannot fall out of their order. The tab
 * chosen is kept under the caller's key (localStorage), the assignees' at first. A row of a state group, an assignee or
 * a label (not the row of none) is a button that chooses its value where the caller says what choosing does (a cycle's
 * or a module's page toggles its filters by it, usePageRowChoices: P5b spec 2.8, P6b spec 2.7); else the rows are
 * plain.
 */
export const ProgressTabs = observer(function ProgressTabs(props: Props) {
  const { progress, projectId, storageKey, choices } = props;
  // hooks
  const { t } = useTranslation();
  const { assignees, labels } = useDistributionRows(progress, projectId);
  const { storedValue: currentTab, setValue: setTab } = useLocalStorage(storageKey, "stat-assignees");
  // derived values
  const selected = Math.max(
    0,
    PROGRESS_STATS.findIndex((stat) => stat.key === currentTab)
  );
  const counted = progress.total_issues - progress.cancelled_issues;
  const panels: Record<ProgressTab, ReactNode> = {
    "stat-states": (
      <>
        <ul>
          {COUNTED.map((group) => (
            <li key={group}>
              <SingleProgressStats
                title={<span className="text-11">{t(STATE_GROUPS[group].i18n_label)}</span>}
                completed={progress[`${group}_issues` as const]}
                total={counted}
                {...choiceOf(choices?.states ?? {}, group)}
              />
            </li>
          ))}
        </ul>
        {progress.cancelled_issues > 0 && (
          <p className="p-1 text-11 text-tertiary">
            {t("project_cycles.active_cycle.cancelled_excluded", { count: progress.cancelled_issues })}
          </p>
        )}
      </>
    ),
    "stat-assignees":
      assignees.length > 0 ? (
        <AssigneeStats rows={assignees} {...choices?.assignees} />
      ) : (
        <NoRows image={emptyMembers}>{t("no_assignee")}</NoRows>
      ),
    "stat-labels":
      labels.length > 0 ? (
        <LabelStats rows={labels} {...choices?.labels} />
      ) : (
        <NoRows image={emptyLabel}>{t("no_labels_yet")}</NoRows>
      ),
  };

  return (
    <Tab.Group
      as="div"
      selectedIndex={selected}
      onChange={(index: number) => setTab(PROGRESS_STATS[index]?.key ?? "stat-assignees")}
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
        {PROGRESS_STATS.map((stat) => (
          <Tab.Panel key={stat.key}>{panels[stat.key]}</Tab.Panel>
        ))}
      </Tab.Panels>
    </Tab.Group>
  );
});

/** A breakdown without a row: no work item of the cycle or the module is counted in it. */
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
