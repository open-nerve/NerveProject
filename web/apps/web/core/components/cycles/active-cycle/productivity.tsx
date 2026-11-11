/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// assets
import darkChartAsset from "@/app/assets/empty-state/active-cycle/chart-dark.webp?url";
import lightChartAsset from "@/app/assets/empty-state/active-cycle/chart-light.webp?url";
// components
import { BurndownChart } from "@/components/core/sidebar/burndown-chart";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
  /** The progress the card shows (cycleCounts' choice). */
  progress: Progress;
};

/**
 * The burndown of the cycle in progress (P5a spec 2.4, 2.5), linked to the cycle's page: its work items pending (of
 * the backlog, unstarted and started groups) and nerve's burndown (BurndownChart); without work items, how to see it.
 * nerve gives no burndown of a span of more than 3660 days: none yet to show.
 */
export function ActiveCycleProductivity({ workspaceSlug, projectId, cycleId, progress }: Props) {
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // derived values
  const title = t("project_cycles.active_cycle.issue_burndown");
  const cyclePage = `/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`;
  const pending = progress.backlog_issues + progress.unstarted_issues + progress.started_issues;

  return (
    <section
      aria-label={title}
      className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1 px-3.5 py-4"
    >
      <div className="relative flex items-center justify-between gap-4">
        <Link to={cyclePage}>
          <h3 className="text-14 font-semibold text-tertiary">{title}</h3>
        </Link>
      </div>

      {progress.total_issues === 0 ? (
        <div className="flex h-full w-full items-center justify-center">
          <SimpleEmptyState
            title={t("active_cycle.empty_state.chart.title")}
            assetPath={resolvedTheme === "light" ? lightChartAsset : darkChartAsset}
          />
        </div>
      ) : (
        <Link to={cyclePage} className="h-full w-full px-2">
          <div className="flex items-center justify-end gap-4 py-1 text-11 text-tertiary">
            <span>{t("project_cycles.active_cycle.pending_work_items", { count: pending })}</span>
          </div>
          <div className="relative h-full">
            {progress.burndown ? (
              <BurndownChart burndown={progress.burndown} />
            ) : (
              <p className="text-13 text-tertiary">{t("no_data_yet")}</p>
            )}
          </div>
        </Link>
      )}
    </section>
  );
}
