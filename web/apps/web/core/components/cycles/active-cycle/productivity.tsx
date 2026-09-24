/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
import { useTheme } from "next-themes";
// plane imports
import { useTranslation } from "@nerve/i18n";
import type { ICycle } from "@nerve/types";
import { Loader } from "@nerve/ui";
// assets
import darkChartAsset from "@/app/assets/empty-state/active-cycle/chart-dark.webp?url";
import lightChartAsset from "@/app/assets/empty-state/active-cycle/chart-light.webp?url";
// components
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";

export type ActiveCycleProductivityProps = {
  workspaceSlug: string;
  projectId: string;
  cycle: ICycle | null;
};

export const ActiveCycleProductivity = observer(function ActiveCycleProductivity(props: ActiveCycleProductivityProps) {
  const { workspaceSlug, projectId, cycle } = props;
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // derived values
  const resolvedPath = resolvedTheme === "light" ? lightChartAsset : darkChartAsset;
  const completionChartDistributionData = cycle?.distribution?.completion_chart || undefined;

  return cycle && completionChartDistributionData ? (
    <div className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1 px-3.5 py-4">
      <div className="relative flex items-center justify-between gap-4">
        <Link to={`/${workspaceSlug}/projects/${projectId}/cycles/${cycle?.id}`}>
          <h3 className="text-14 font-semibold text-tertiary">{t("project_cycles.active_cycle.issue_burndown")}</h3>
        </Link>
      </div>

      <Link to={`/${workspaceSlug}/projects/${projectId}/cycles/${cycle?.id}`}>
        {cycle.total_issues > 0 ? (
          <>
            <div className="h-full w-full px-2">
              <div className="flex items-center justify-end gap-4 py-1 text-11 text-tertiary">
                <span>{`Pending work items - ${cycle.backlog_issues + cycle.unstarted_issues + cycle.started_issues}`}</span>
              </div>

              <div className="relative h-full">
                {completionChartDistributionData && (
                  <Fragment>
                    <ProgressChart
                      distribution={completionChartDistributionData}
                      totalIssues={cycle.total_issues || 0}
                      plotTitle={"work items"}
                    />
                  </Fragment>
                )}
              </div>
            </div>
          </>
        ) : (
          <>
            <div className="flex h-full w-full items-center justify-center">
              <SimpleEmptyState title={t("active_cycle.empty_state.chart.title")} assetPath={resolvedPath} />
            </div>
          </>
        )}
      </Link>
    </div>
  ) : (
    <Loader className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1">
      <Loader.Item width="100%" height="100%" />
    </Loader>
  );
});
