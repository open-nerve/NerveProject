/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The cycles of the tests of the planning stores (M6 design 3.1–3.8), against a fake nerve: a cycle as nerve's lists
// give it and its progress as nerve reads it, the requests of the cycles, and the fetches that load them into a store.
// The cycles themselves are fake-projects.ts's cycleOf.

import type { Cycle, CycleListItem, CycleProgress, Progress, Project } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { answered } from "@/lib/auth/fake-nerve";
import { web } from "@/store/fake-project-lists";
import type { ICycleStore } from "@/store/planning/cycle.store";

/** A cycle as a list of nerve's gives it: of no work items, with no assignees, not frozen, unless fields say so. */
export function rowOf(cycle: Cycle, fields: Partial<CycleListItem> = {}): CycleListItem {
  return {
    ...cycle,
    total_issues: 0,
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
    assignee_ids: [],
    progress_snapshot: null,
    ...fields,
  };
}

/** A cycle's progress as nerve reads it: its status, live of no work items unless live says so, not frozen. */
export function progressOf(cycle: Cycle, live: Partial<Progress> = {}): CycleProgress {
  return {
    status: cycle.status,
    live: {
      total_issues: 0,
      backlog_issues: 0,
      unstarted_issues: 0,
      started_issues: 0,
      completed_issues: 0,
      cancelled_issues: 0,
      assignees: [],
      labels: [],
      burndown: null,
      ...live,
    },
    progress_snapshot: null,
  };
}

/**
 * The requests of the cycles: a project's lists (the archived one with archived=true), where it creates its cycles; a
 * cycle, which is read, changed and deleted there; its progress, its archiving, unarchiving and transfer.
 */
export const cycleRequests = {
  list: (project: Pick<Project, "id">) => `/api/v0/projects/${project.id}/cycles`,
  at: (cycle: Pick<Cycle, "id">) => `/api/v0/cycles/${cycle.id}`,
  progress: (cycle: Pick<Cycle, "id">) => `/api/v0/cycles/${cycle.id}/progress`,
  archive: (cycle: Pick<Cycle, "id">) => `/api/v0/cycles/${cycle.id}/archive`,
  unarchive: (cycle: Pick<Cycle, "id">) => `/api/v0/cycles/${cycle.id}/unarchive`,
  transfer: (cycle: Pick<Cycle, "id">) => `/api/v0/cycles/${cycle.id}/transfer-issues`,
};

/** The store fetches the project's cycles that are not archived, web's unless it says, and nerve lists these. */
export function loadCycles(nerve: FakeNerve, store: ICycleStore, rows: CycleListItem[], project: Project = web) {
  const path = cycleRequests.list(project);
  return answered(nerve, () => store.fetchCycles(project.id), ["GET", path], { data: rows }, "the cycles");
}

/** The store fetches the project's archived cycles, web's unless it says, and nerve lists these. */
export function loadArchivedCycles(
  nerve: FakeNerve,
  store: ICycleStore,
  rows: CycleListItem[],
  project: Project = web
) {
  const path = cycleRequests.list(project);
  return answered(nerve, () => store.fetchArchivedCycles(project.id), ["GET", path], { data: rows }, "the archived");
}

/** The store reads the cycle alone, and nerve gives it as read. */
export function loadCycle(nerve: FakeNerve, store: ICycleStore, read: Cycle) {
  return answered(nerve, () => store.fetchCycle(read.id), ["GET", cycleRequests.at(read)], read, "the cycle");
}

/** The store reads the cycle's progress, and nerve gives progress. */
export function loadProgress(nerve: FakeNerve, store: ICycleStore, cycle: Cycle, progress: CycleProgress) {
  const path = cycleRequests.progress(cycle);
  return answered(nerve, () => store.fetchProgress(cycle.id), ["GET", path], progress, "the progress");
}
