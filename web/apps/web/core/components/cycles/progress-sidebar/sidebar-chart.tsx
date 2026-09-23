/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { observer } from "mobx-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Loader } from "@plane/ui";
import { getDate } from "@plane/utils";
// components
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { validateCycleSnapshot } from "@/components/cycles/progress-sidebar/issue-progress";
// hooks
import { useCycle } from "@/hooks/store/use-cycle";

type ProgressChartProps = {
  cycleId: string;
};
export const SidebarChart = observer(function SidebarChart(props: ProgressChartProps) {
  const { cycleId } = props;

  // hooks
  const { getCycleById } = useCycle();
  const { t } = useTranslation();

  // derived data
  const cycleDetails = validateCycleSnapshot(getCycleById(cycleId));
  const cycleStartDate = getDate(cycleDetails?.start_date);
  const cycleEndDate = getDate(cycleDetails?.end_date);
  const totalIssues = cycleDetails?.total_issues || 0;
  const completionChartDistributionData = cycleDetails?.distribution?.completion_chart || undefined;

  return (
    <div>
      <div className="py-4">
        <div>
          {cycleStartDate && cycleEndDate && completionChartDistributionData ? (
            <Fragment>
              <ProgressChart
                distribution={completionChartDistributionData}
                totalIssues={totalIssues}
                plotTitle={t("work_items")}
              />
            </Fragment>
          ) : (
            <Loader className="mt-4 h-[160px] w-full">
              <Loader.Item width="100%" height="100%" />
            </Loader>
          )}
        </div>
      </div>
    </div>
  );
});
