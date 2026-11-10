/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import type { Burndown } from "@nerve/api-client";
import type { TAreaChartProps } from "@nerve/types";
import { BurndownChart } from "./burndown-chart";

// What the burndown chart hands the area chart (P5a spec 2.4): the area chart is a stand-in that keeps the props it
// was given, and a text is its key (fake-i18n.ts). nerve's burndown has a point for every day of the cycle, the work
// left at the day's end, null for a day after today (P4 spec 2.1).

const chart = vi.hoisted((): { props: TAreaChartProps<string, string> | undefined } => ({ props: undefined }));
vi.mock("@nerve/propel/charts/area-chart", () => ({
  AreaChart: (props: TAreaChartProps<string, string>) => {
    chart.props = props;
    return null;
  },
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** A burndown of the days given, each with the work left at its end (null after today), from scope work items. */
function burndownOf(scope: number, days: [string, number | null][]): Burndown {
  return {
    start_date: days[0][0],
    end_date: days[days.length - 1][0],
    scope,
    points: days.map(([date, remaining]) => ({ date, remaining })),
  };
}

/** What the area chart was given to draw for the burndown. */
function drawn(burndown: Burndown): TAreaChartProps<string, string> {
  chart.props = undefined;
  renderToStaticMarkup(<BurndownChart burndown={burndown} />);
  if (!chart.props) throw new Error("the burndown chart drew no area chart");
  return chart.props;
}

/** The chart's days: each day's label, the work left (null: no point) and the ideal line's value. */
const daysOf = (burndown: Burndown) => drawn(burndown).data;

// A zone west of UTC, where a "YYYY-MM-DD" read as a time (UTC's midnight) falls on the day before: a day read
// through new Date("YYYY-MM-DD") shows a day early here. The zone is this file's alone (vi.stubEnv); the CI's is UTC.
beforeAll(() => {
  vi.stubEnv("TZ", "America/Los_Angeles");
});
afterAll(() => {
  vi.unstubAllEnvs();
});

describe("BurndownChart", () => {
  it("runs in a zone west of UTC", () => {
    expect(new Date(2026, 10, 11).getTimezoneOffset()).toBe(480);
  });

  it("draws a one-day cycle: its work left, and an ideal of 0 on its only day", () => {
    expect(daysOf(burndownOf(3, [["2026-11-11", 3]]))).toEqual([{ date: "Nov 11", remaining: 3, ideal: 0 }]);
  });

  it("draws two days: the ideal falls from the scope on the first to 0 on the last", () => {
    expect(
      daysOf(
        burndownOf(4, [
          ["2026-11-10", 4],
          ["2026-11-11", 1],
        ])
      )
    ).toEqual([
      { date: "Nov 10", remaining: 4, ideal: 4 },
      { date: "Nov 11", remaining: 1, ideal: 0 },
    ]);
  });

  it("leaves the days after today without a point, never at 0, the ideal falling straight", () => {
    expect(
      daysOf(
        burndownOf(8, [
          ["2026-10-30", 8],
          ["2026-10-31", 7],
          ["2026-11-01", 5],
          ["2026-11-02", null],
          ["2026-11-03", null],
        ])
      )
    ).toEqual([
      { date: "Oct 30", remaining: 8, ideal: 8 },
      { date: "Oct 31", remaining: 7, ideal: 6 },
      { date: "Nov 01", remaining: 5, ideal: 4 },
      { date: "Nov 02", remaining: null, ideal: 2 },
      { date: "Nov 03", remaining: null, ideal: 0 },
    ]);
  });

  it("draws no work left at all while the cycle has not started (every day after today)", () => {
    expect(
      daysOf(
        burndownOf(6, [
          ["2026-12-01", null],
          ["2026-12-02", null],
          ["2026-12-03", null],
        ])
      )
    ).toEqual([
      { date: "Dec 01", remaining: null, ideal: 6 },
      { date: "Dec 02", remaining: null, ideal: 3 },
      { date: "Dec 03", remaining: null, ideal: 0 },
    ]);
  });

  it("gives the chart finite numbers alone, whatever the burndown's shape", () => {
    const shapes = [
      burndownOf(3, [["2026-11-11", 3]]),
      burndownOf(0, [["2026-11-11", 0]]),
      burndownOf(0, [["2026-11-11", null]]),
      burndownOf(5, [
        ["2026-11-10", 5],
        ["2026-11-11", null],
      ]),
      burndownOf(7, [
        ["2026-11-09", null],
        ["2026-11-10", null],
        ["2026-11-11", null],
      ]),
    ];
    for (const shape of shapes) {
      for (const day of daysOf(shape)) {
        expect(Number.isFinite(day.ideal), `the ideal of ${String(day.date)}`).toBe(true);
        expect(day.remaining === null || Number.isFinite(day.remaining), `the work left on ${String(day.date)}`).toBe(
          true
        );
      }
    }
  });

  it("draws no day of a burndown without one", () => {
    // nerve gives none (a cycle's days give one point at least; a draft's burndown is null), but the chart holds.
    expect(daysOf({ start_date: "2026-11-11", end_date: "2026-11-11", scope: 2, points: [] })).toEqual([]);
  });

  it("names its lines and axes in the page's language, the days along the x axis, the work left up the y axis", () => {
    const props = drawn(burndownOf(3, [["2026-11-11", 3]]));
    expect(props.areas.map(({ key, label }) => [key, label])).toEqual([
      ["remaining", "burndown.remaining"],
      ["ideal", "burndown.ideal"],
    ]);
    expect(props.xAxis).toMatchObject({ key: "date", label: "common.date" });
    expect(props.yAxis).toMatchObject({ key: "remaining", label: "work_items" });
  });
});
