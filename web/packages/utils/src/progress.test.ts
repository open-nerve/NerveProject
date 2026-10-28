/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { ICycle } from "@nerve/types";
import { calculateCycleProgress } from "./cycle";

// Cycle progress is a work-item count (M1 design 3.4). These tests pin the percentage a cycle reports.

const cycle = (fields: Partial<ICycle>) => fields as ICycle;

describe("calculateCycleProgress", () => {
  it("is zero without a cycle", () => {
    expect(calculateCycleProgress(undefined)).toBe(0);
  });

  it("is zero for an empty cycle", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 0, completed_issues: 0, cancelled_issues: 0 }))).toBe(0);
  });

  it("leaves the cancelled work items out of the total", () => {
    // 3 of the 8 remaining work items are done: 6 of 10 were not cancelled.
    expect(calculateCycleProgress(cycle({ total_issues: 10, completed_issues: 3, cancelled_issues: 2 }))).toBe(38);
  });

  it("counts the started work items as well when asked to", () => {
    expect(
      calculateCycleProgress(
        cycle({ total_issues: 10, completed_issues: 3, cancelled_issues: 2, started_issues: 1 }),
        true
      )
    ).toBe(50);
  });

  it("reports 100 once everything that was not cancelled is done", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 10, completed_issues: 8, cancelled_issues: 2 }))).toBe(100);
  });

  it("reports zero when every work item was cancelled", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 4, completed_issues: 0, cancelled_issues: 4 }))).toBe(0);
  });

  it("prefers the snapshot of a completed cycle over the live counts", () => {
    const completed = cycle({
      total_issues: 100,
      completed_issues: 0,
      cancelled_issues: 0,
      progress_snapshot: { total_issues: 4, completed_issues: 2, cancelled_issues: 0 } as ICycle["progress_snapshot"],
    });
    expect(calculateCycleProgress(completed)).toBe(50);
  });
});
