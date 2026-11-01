/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, describe, expect, it, vi } from "vitest";
import { getFilteredWorkItems } from "./base-issues-utils";
import { issueOf } from "../fake-issues";
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllEnvs();
});
describe.each([
  ["UTC", 0],
  ["Asia/Shanghai", -480],
  ["America/New_York", 240],
] as const)("sub-item calendar dates in %s", (zone, offset) => {
  it.each([
    ["1_weeks;after;fromnow", "2026-10-08", "2026-10-07"],
    ["2_weeks;after;fromnow", "2026-10-15", "2026-10-14"],
    ["1_months;after;fromnow", "2026-11-01", "2026-10-31"],
    ["2_months;after;fromnow", "2026-12-01", "2026-11-30"],
    ["1_weeks;before;fromnow", "2026-10-08", "2026-10-09"],
    ["2_weeks;before;fromnow", "2026-10-15", "2026-10-16"],
    ["1_months;before;fromnow", "2026-11-01", "2026-11-02"],
    ["2026-10-08;after", "2026-10-08", "2026-10-07"],
    ["2026-10-08;before", "2026-10-08", "2026-10-09"],
  ])("subIssueRelativeDatesStayLocal %s", (condition, included, excluded) => {
    vi.stubEnv("TZ", zone);
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 9, 1, 12));
    expect(new Date().getTimezoneOffset()).toBe(offset);
    const rows = [
      { ...issueOf("yes"), target_date: included },
      { ...issueOf("no"), target_date: excluded },
    ];
    expect(getFilteredWorkItems(rows, { target_date: [condition] }).map((issue) => issue.id)).toEqual(["yes"]);
  });
});
