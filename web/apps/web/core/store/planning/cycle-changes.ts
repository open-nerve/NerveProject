/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
// nerve imports
import type {
  Cycle,
  CycleCreate,
  CycleListItem,
  CycleProgress,
  CycleTransferResult,
  CycleUpdate,
} from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change, ReconciledByKey } from "@/lib/reconciled";
import { dropped, prepended } from "@/lib/reconciled";
// services
import type { CyclesService } from "@/services/planning/cycles.service";

export interface ICycleChanges {
  // changes
  create: (projectId: string, body: CycleCreate) => Promise<Cycle>;
  update: (cycleId: string, body: CycleUpdate) => Promise<Cycle>;
  remove: (cycleId: string) => Promise<void>;
  archive: (cycleId: string) => Promise<Cycle>;
  unarchive: (cycleId: string) => Promise<Cycle>;
  transfer: (cycleId: string, targetCycleId: string) => Promise<CycleTransferResult>;
}

/** What the changes write nerve's answers into, read their cycle from and fetch again: the cycle store's (CycleStore). */
type HeldCycles = {
  readonly unarchived: ReconciledByKey<CycleListItem[]>;
  readonly archived: ReconciledByKey<CycleListItem[]>;
  readonly details: ReconciledByKey<Cycle | null>;
  readonly progress: ReconciledByKey<CycleProgress>;
  getCycleById: (cycleId: string) => Cycle | undefined;
  fetchCycles: (projectId: string) => Promise<unknown>;
  fetchProgress: (cycleId: string) => Promise<unknown>;
};

/** A cycle nerve has just created, as its project's list gives it: it has no work items yet, so these are exact. */
const newRow = (cycle: Cycle): CycleListItem => ({
  ...cycle,
  total_issues: 0,
  backlog_issues: 0,
  unstarted_issues: 0,
  started_issues: 0,
  completed_issues: 0,
  cancelled_issues: 0,
  assignee_ids: [],
  progress_snapshot: null,
});

/** The cycle's row, where the list has it, with its fields as nerve answered them: its counts stay its own (R9). */
const written =
  (cycle: Cycle): Change<CycleListItem[]> =>
  (list) =>
    list.map((row) => (row.id === cycle.id ? { ...row, ...cycle } : row));

/**
 * The cycle moved into a list, with the row it left (its counts): first, as nerve lists it, the latest archived first
 * in the archive; where the list has it already (a repeat), written in its place. Without a row it adds nothing: the
 * list's next fetch brings it.
 */
const landed =
  (cycle: Cycle, row: CycleListItem | undefined): Change<CycleListItem[]> =>
  (list) =>
    !row || list.some((listed) => listed.id === cycle.id) ? written(cycle)(list) : [{ ...row, ...cycle }, ...list];

/**
 * The changes of the cycles (P5a spec 2.2), one at a time (v0 design 7.7). Once nerve confirms a change, its answer is
 * written into every copy of the cycle the store holds: its project's lists' rows, its own read, its progress's status;
 * nerve's answers have no counts, so a row keeps its own (R9), and a transfer's moved counts are fetched again, never
 * reckoned. A deletion or a transfer finds its cycle in its turn. A refusal fails the change and writes nothing.
 */
export class CycleChanges implements ICycleChanges {
  /** The changes of the cycles, sent one at a time. */
  private readonly queue = oneAtATime();

  constructor(
    private readonly cycles: HeldCycles,
    private readonly service: CyclesService
  ) {}

  /**
   * @description creates a cycle: first of its project's list, as nerve lists it (its sort order the lowest), of no
   * work items. Fails, changing nothing, when nerve refuses.
   */
  create = (projectId: string, body: CycleCreate): Promise<Cycle> =>
    this.queue(async () => {
      const cycle = await this.service.create(projectId, body);
      this.cycles.unarchived.confirm(cycle.project_id, prepended([newRow(cycle)]));
      return cycle;
    });

  /** @description changes a cycle; the store then shows nerve's answer. Fails, changing nothing, when refused. */
  update = (cycleId: string, body: CycleUpdate): Promise<Cycle> =>
    this.queue(async () => {
      const cycle = await this.service.update(cycleId, body);
      runInAction(() => {
        this.cycles.unarchived.confirm(cycle.project_id, written(cycle));
        this.cycles.archived.confirm(cycle.project_id, written(cycle));
        this.answered(cycle);
      });
      return cycle;
    });

  /**
   * @description archives an ended cycle: its row moves first into its project's archive. One archived already is
   * answered as nerve holds it, which the store writes. Fails, changing nothing, when refused.
   */
  archive = (cycleId: string): Promise<Cycle> =>
    this.queue(async () => this.moved(await this.service.archive(cycleId), "unarchived", "archived"));

  /** @description unarchives a cycle: its row moves first into its project's list, as archive */
  unarchive = (cycleId: string): Promise<Cycle> =>
    this.queue(async () => this.moved(await this.service.unarchive(cycleId), "archived", "unarchived"));

  /**
   * @description deletes a cycle, which the store then no longer gives, nor its progress. Fails, changing nothing,
   * when nerve refuses, or asking nerve nothing when the store does not give the cycle in its turn.
   */
  remove = (cycleId: string): Promise<void> =>
    this.inTurn(cycleId, async (cycle) => {
      await this.service.remove(cycle.id);
      runInAction(() => {
        this.cycles.unarchived.confirm(cycle.project_id, dropped(cycle.id));
        this.cycles.archived.confirm(cycle.project_id, dropped(cycle.id));
        this.cycles.details.confirm(cycle.id, () => null);
      });
    });

  /**
   * @description moves an ended cycle's unfinished work items to the target cycle; gives how many moved. Then it
   * fetches again both cycles' progress and their project's list, whose counts changed (the queue does not wait for
   * them; a failed fetch leaves what the store had). Fails, changing nothing, as remove.
   */
  transfer = (cycleId: string, targetCycleId: string): Promise<CycleTransferResult> =>
    this.inTurn(cycleId, async (cycle) => {
      const result = await this.service.transfer(cycle.id, targetCycleId);
      void Promise.allSettled([
        this.cycles.fetchProgress(cycle.id),
        this.cycles.fetchProgress(targetCycleId),
        this.cycles.fetchCycles(cycle.project_id),
      ]);
      return result;
    });

  /** Sends a change of a cycle in its turn, with the cycle as the store gives it then; fails when it gives none. */
  private inTurn<T>(cycleId: string, send: (cycle: Cycle) => Promise<T>): Promise<T> {
    return this.queue(() => send(this.shown(cycleId)));
  }

  /** The cycle as the store gives it; fails when it gives none. */
  private shown(cycleId: string): Cycle {
    const cycle = this.cycles.getCycleById(cycleId);
    if (!cycle) throw new Error("Cycle not found");
    return cycle;
  }

  /** Moves the cycle nerve archived or unarchived from one of its project's lists to the other, with its row. */
  private moved(cycle: Cycle, from: "unarchived" | "archived", to: "unarchived" | "archived"): Cycle {
    runInAction(() => {
      const row = this.cycles[from].get(cycle.project_id)?.find((listed) => listed.id === cycle.id);
      this.cycles[from].confirm(cycle.project_id, dropped(cycle.id));
      this.cycles[to].confirm(cycle.project_id, landed(cycle, row));
      this.answered(cycle);
    });
    return cycle;
  }

  /** The cycle's own read and its progress's status, as nerve answered the cycle; a deleted one's stays deleted. */
  private answered(cycle: Cycle): void {
    this.cycles.details.confirm(cycle.id, (read) => read && cycle);
    this.cycles.progress.confirm(cycle.id, (progress) => ({ ...progress, status: cycle.status }));
  }
}
