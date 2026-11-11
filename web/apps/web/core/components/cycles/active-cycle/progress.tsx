/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTheme } from "next-themes";
// nerve imports
import { LinearProgress } from "@makeplane/propel/components/linear-progress";
import type { Progress } from "@nerve/api-client";
import { STATE_GROUPS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { progressPercent } from "@nerve/utils";
// assets
import darkProgressAsset from "@/app/assets/empty-state/active-cycle/progress-dark.webp?url";
import lightProgressAsset from "@/app/assets/empty-state/active-cycle/progress-light.webp?url";
// components
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";

/** The state groups whose work items a cycle's progress counts, completed first; the cancelled ones it does not. */
const COUNTED = ["completed", "started", "unstarted", "backlog"] as const;

type Props = {
  /** The progress the card shows (cycleCounts' choice). */
  progress: Progress;
};

/**
 * The progress of the cycle in progress (P5a spec 2.5): the completed of its work items not cancelled (progressPercent),
 * the work items of each state group it counts that has any, and the cancelled ones said apart; without work items,
 * how to see it. A row does not set the page's filters (P5b).
 */
export function ActiveCycleProgress({ progress }: Props) {
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // derived values
  const title = t("project_cycles.active_cycle.progress");

  return (
    <section
      aria-label={title}
      className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1 px-3.5 py-4"
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-4">
          <h3 className="text-14 font-semibold text-tertiary">{title}</h3>
          {progress.total_issues > 0 && (
            <span className="flex gap-1 rounded-xs px-3 py-1 text-13 font-medium whitespace-nowrap text-placeholder">
              {t("project_cycles.active_cycle.work_items_completed", {
                completed: progress.completed_issues,
                total: progress.total_issues - progress.cancelled_issues,
              })}
            </span>
          )}
        </div>
        {progress.total_issues > 0 && (
          <LinearProgress
            value={progressPercent(progress)}
            size="md"
            variant="brand"
            showValue={false}
            aria-label={title}
          />
        )}
      </div>

      {progress.total_issues > 0 ? (
        <div className="flex flex-col gap-5">
          <ul className="flex flex-col gap-5">
            {COUNTED.filter((group) => progress[`${group}_issues`] > 0).map((group) => {
              const count = progress[`${group}_issues`];
              return (
                <li key={group} className="flex items-center justify-between gap-2 text-13">
                  <div className="flex items-center gap-1.5">
                    <span
                      className="block h-3 w-3 rounded-full"
                      style={{ backgroundColor: STATE_GROUPS[group].color }}
                      aria-hidden="true"
                    />
                    <span className="font-medium text-tertiary">{t(STATE_GROUPS[group].i18n_label)}</span>
                  </div>
                  <span className="text-tertiary">{`${count} ${t("issue.label", { count })}`}</span>
                </li>
              );
            })}
          </ul>
          {progress.cancelled_issues > 0 && (
            <span className="flex items-center gap-2 text-13 text-tertiary">
              {t("project_cycles.active_cycle.cancelled_excluded", { count: progress.cancelled_issues })}
            </span>
          )}
        </div>
      ) : (
        <div className="flex h-full w-full items-center justify-center">
          <SimpleEmptyState
            title={t("active_cycle.empty_state.progress.title")}
            assetPath={resolvedTheme === "light" ? lightProgressAsset : darkProgressAsset}
          />
        </div>
      )}
    </section>
  );
}
