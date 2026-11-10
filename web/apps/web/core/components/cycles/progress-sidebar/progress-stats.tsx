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
import { cn } from "@nerve/utils";
// assets
import emptyLabel from "@/app/assets/empty-state/empty_label.svg?url";
import emptyMembers from "@/app/assets/empty-state/empty_members.svg?url";
import userImage from "@/app/assets/user.png?url";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
import { PROGRESS_STATS } from "@/components/core/sidebar/progress-stats/shared";
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import useLocalStorage from "@/hooks/use-local-storage";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// local imports
import { assigneeRows, labelRows } from "../distribution-rows";

/** The state groups whose work items a cycle's progress counts, in their order; the cancelled ones it does not. */
const COUNTED = ["backlog", "unstarted", "started", "completed"] as const;

type Props = {
  cycleId: string;
  projectId: string;
  /** The progress the sidebar shows (cycleCounts' choice): its counts, its assignees and its labels. */
  progress: Progress;
};

/**
 * The breakdowns of a cycle's progress (P5a spec 2.4, 2.5), each a tab: by state, each group's share of the work items
 * not cancelled (progressPercent's, the cancelled said apart); by assignee and by label, as nerve counts them, named by
 * the project's members and labels (distribution-rows.ts). The tab chosen is kept for the cycle (localStorage); a row
 * does not set the page's filters (P5b).
 */
export const CycleProgressStats = observer(function CycleProgressStats(props: Props) {
  const { cycleId, projectId, progress } = props;
  // hooks
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  const {
    project: { getProjectMemberDetails },
  } = useMember();
  const { getProjectLabels } = useLabel();
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
  const assignees = assigneeRows(progress.assignees, (id) => getProjectMemberDetails(id, projectId)?.member, t);
  const labels = labelRows(progress.labels, getProjectLabels(projectId) ?? [], t);

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
            <ul>
              {assignees.map((row) => (
                <li key={row.assignee_id ?? "none"}>
                  <SingleProgressStats
                    title={
                      <div className="flex items-center gap-2">
                        {row.avatarOf === null ? (
                          <img src={userImage} className="size-4 rounded-full object-cover" alt="" aria-hidden="true" />
                        ) : (
                          <ProfileAvatar
                            alt={row.name}
                            fallback={row.name[0]?.toUpperCase()}
                            source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: row.avatarOf }}
                            size="xs"
                          />
                        )}
                        <span>{row.name}</span>
                      </div>
                    }
                    completed={row.completed_issues}
                    total={row.total_issues}
                  />
                </li>
              ))}
            </ul>
          ) : (
            <NoRows image={emptyMembers}>{t("no_assignee")}</NoRows>
          )}
        </Tab.Panel>
        <Tab.Panel>
          {labels.length > 0 ? (
            <ul>
              {labels.map((row) => (
                <li key={row.label_id ?? "none"}>
                  <SingleProgressStats
                    title={
                      <div className="flex items-center gap-2 truncate">
                        <span
                          className="block h-3 w-3 flex-shrink-0 rounded-full"
                          style={{ backgroundColor: row.color ?? undefined }}
                          aria-hidden="true"
                        />
                        <span className="truncate text-11 text-ellipsis">{row.name}</span>
                      </div>
                    }
                    completed={row.completed_issues}
                    total={row.total_issues}
                  />
                </li>
              ))}
            </ul>
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
