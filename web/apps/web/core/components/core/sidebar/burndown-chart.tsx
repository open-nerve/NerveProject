/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { AreaChart } from "@nerve/propel/charts/area-chart";
import type { Burndown } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import type { TChartData } from "@nerve/types";
import { renderFormattedDateWithoutYear } from "@nerve/utils";

type Props = {
  burndown: Burndown;
};

/** A day of the chart: its calendar day, the work left at its end (null after today: no point), the ideal line's. */
type Day = { date: string; remaining: number | null; ideal: number };

/**
 * The ideal line's value on the day at index of days: it falls straight from the scope on the first day to 0 on the
 * last; a cycle of one day is on its last day, 0.
 */
const idealOn = (scope: number, index: number, days: number): number =>
  days > 1 ? scope * (1 - index / (days - 1)) : 0;

/**
 * A cycle's (or a module's) burndown as nerve gives it (P5a spec 2.4): its days along the x axis, in nerve's order,
 * each its calendar day (no time zone's: getDate reads "YYYY-MM-DD" as the local day); the work left on each, where a
 * day after today has none to draw (null, never 0); and the ideal line from the scope to 0. Every number it draws is
 * finite. nerve gives no burndown without a day (a draft's is null), so the page shows the chart only with one.
 */
export function BurndownChart({ burndown }: Props) {
  const { t } = useTranslation();
  const days = burndown.points.length;
  const data: TChartData<"date", "remaining" | "ideal">[] = burndown.points.map(
    (point, index): Day => ({
      date: renderFormattedDateWithoutYear(point.date),
      remaining: point.remaining,
      ideal: idealOn(burndown.scope, index, days),
    })
  );

  return (
    <div className="flex w-full items-center justify-center">
      <AreaChart
        data={data}
        areas={[
          {
            key: "remaining",
            label: t("burndown.remaining"),
            strokeColor: "#3F76FF",
            fill: "#3F76FF33",
            fillOpacity: 1,
            showDot: true,
            smoothCurves: true,
            strokeOpacity: 1,
            stackId: "remaining",
          },
          {
            key: "ideal",
            label: t("burndown.ideal"),
            strokeColor: "#A9BBD0",
            fill: "#A9BBD0",
            fillOpacity: 0,
            showDot: true,
            smoothCurves: true,
            strokeOpacity: 1,
            stackId: "ideal",
            style: {
              strokeDasharray: "6, 3",
              strokeWidth: 1,
            },
          },
        ]}
        xAxis={{ key: "date", label: t("common.date") }}
        yAxis={{ key: "remaining", label: t("work_items") }}
        margin={{ bottom: 30 }}
        className="h-[370px] w-full"
        legend={{
          align: "center",
          verticalAlign: "bottom",
          layout: "horizontal",
          wrapperStyles: {
            marginTop: 20,
          },
        }}
      />
    </div>
  );
}
