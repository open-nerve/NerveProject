/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import type { ICycle, IModule, IState, TIssue } from "@nerve/types";
import { calculateCycleProgress } from "./cycle";
import { getDistributionPathsPostUpdate, updateDistribution } from "./distribution-update";

// Cycle and module progress is a work-item count and nothing else (M1 design 3.4). These tests pin
// the two calculations the product depends on: the percentage a cycle reports, and the optimistic
// update that runs when a work item is created, completed, cancelled or reopened.

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

const STATE_MAP: Record<string, IState> = {
  backlog: { id: "backlog", group: "backlog" } as IState,
  started: { id: "started", group: "started" } as IState,
  done: { id: "done", group: "completed" } as IState,
  cancelled: { id: "cancelled", group: "cancelled" } as IState,
};

const workItem = (stateId: string, fields: Partial<TIssue> = {}) =>
  ({
    id: "work-item-1",
    state_id: stateId,
    assignee_ids: [],
    label_ids: [],
    completed_at: null,
    ...fields,
  }) as TIssue;

const emptyCycle = () =>
  ({
    total_issues: 0,
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
    distribution: { assignees: [], labels: [], completion_chart: {} },
  }) as unknown as ICycle;

describe("the optimistic count update", () => {
  it("adds a new work item to the total and to its own state group", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("backlog"), STATE_MAP));
    expect(target.total_issues).toBe(1);
    expect(target.backlog_issues).toBe(1);
    expect(target.completed_issues).toBe(0);
  });

  it("moves the count between state groups when a work item is completed", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("started"), STATE_MAP));
    updateDistribution(
      target,
      getDistributionPathsPostUpdate(workItem("started"), workItem("done", { completed_at: "2026-01-02" }), STATE_MAP)
    );
    expect(target.total_issues).toBe(1);
    expect(target.started_issues).toBe(0);
    expect(target.completed_issues).toBe(1);
  });

  it("moves it back when the work item is reopened", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("done"), STATE_MAP));
    updateDistribution(target, getDistributionPathsPostUpdate(workItem("done"), workItem("started"), STATE_MAP));
    expect(target.completed_issues).toBe(0);
    expect(target.started_issues).toBe(1);
    expect(target.total_issues).toBe(1);
  });

  it("takes a removed work item out of the total", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("cancelled"), STATE_MAP));
    expect(target.total_issues).toBe(1);
    expect(target.cancelled_issues).toBe(1);
    updateDistribution(target, getDistributionPathsPostUpdate(workItem("cancelled"), undefined, STATE_MAP));
    expect(target.total_issues).toBe(0);
    expect(target.cancelled_issues).toBe(0);
  });

  it("keeps the assignee and label counts of a module in step", () => {
    const target = {
      total_issues: 0,
      backlog_issues: 0,
      unstarted_issues: 0,
      started_issues: 0,
      completed_issues: 0,
      cancelled_issues: 0,
      distribution: {
        assignees: [{ assignee_id: "user-1", completed_issues: 0, pending_issues: 0, total_issues: 0 }],
        labels: [{ label_id: "label-1", completed_issues: 0, pending_issues: 0, total_issues: 0 }],
        completion_chart: {},
      },
    } as unknown as IModule;
    const item = workItem("started", { assignee_ids: ["user-1"], label_ids: ["label-1"] });
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, item, STATE_MAP));
    expect(target.distribution?.assignees[0]).toMatchObject({
      pending_issues: 1,
      completed_issues: 0,
      total_issues: 1,
    });
    expect(target.distribution?.labels[0]).toMatchObject({ pending_issues: 1, completed_issues: 0, total_issues: 1 });

    const done = workItem("done", { assignee_ids: ["user-1"], label_ids: ["label-1"], completed_at: "2026-01-02" });
    updateDistribution(target, getDistributionPathsPostUpdate(item, done, STATE_MAP));
    expect(target.distribution?.assignees[0]).toMatchObject({
      pending_issues: 0,
      completed_issues: 1,
      total_issues: 1,
    });
    expect(target.distribution?.labels[0]).toMatchObject({ pending_issues: 0, completed_issues: 1, total_issues: 1 });
  });

  it("takes the completed work item off the burn-down chart on the day it was completed", () => {
    const target = emptyCycle();
    // the chart only carries the days the cycle already knows about, so seed the day under test
    target.distribution!.completion_chart = { "2026-01-02": 5 };
    const started = workItem("started");
    const done = workItem("done", { completed_at: "2026-01-02T10:00:00Z" });
    updateDistribution(target, getDistributionPathsPostUpdate(started, done, STATE_MAP));
    expect(target.distribution?.completion_chart["2026-01-02"]).toBe(4);
  });
});
