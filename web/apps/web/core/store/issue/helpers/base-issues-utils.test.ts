/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, expect, it, vi } from "vitest";
import { getFilteredWorkItems } from "./base-issues-utils";
import { issueOf } from "../fake-issues";
afterEach(() => vi.useRealTimers());
it.each([
  ["1_weeks;after;fromnow", "2026-10-08", "2026-10-07"],
  ["2_weeks;after;fromnow", "2026-10-15", "2026-10-14"],
  ["1_months;after;fromnow", "2026-11-01", "2026-10-31"],
  ["2_months;after;fromnow", "2026-12-01", "2026-11-30"],
  ["1_weeks;before;fromnow", "2026-10-08", "2026-10-09"],
  ["2_weeks;before;fromnow", "2026-10-15", "2026-10-16"],
  ["1_months;before;fromnow", "2026-11-01", "2026-11-02"],
])("subIssueRelativeDatesStayLocal %s", (condition, included, excluded) => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-10-01T12:00:00Z"));
  const rows = [
    { ...issueOf("yes"), target_date: included },
    { ...issueOf("no"), target_date: excluded },
  ];
  expect(getFilteredWorkItems(rows, { target_date: [condition] }).map((issue) => issue.id)).toEqual(["yes"]);
});
