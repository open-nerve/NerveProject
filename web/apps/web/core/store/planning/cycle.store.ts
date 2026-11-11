/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { maxBy } from "lodash-es";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Cycle, CycleListItem, CycleProgress, Project } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { CyclesService } from "@/services/planning/cycles.service";
// store
import type { RootStore } from "@/store/root.store";
import type { ICycleChanges } from "./cycle-changes";
import { CycleChanges } from "./cycle-changes";

/** A copy of a cycle the store holds, and when nerve answered it (Reconciled.answeredAt). */
type Copy<T extends Cycle> = { cycle: T; answeredAt: number };

export interface ICycleStore extends ICycleChanges {
  // computed actions
  getCycleById: (cycleId: string | null | undefined) => Cycle | undefined;
  getCycleRow: (cycleId: string | null | undefined) => CycleListItem | undefined;
  getProjectCycleIds: (projectId: string | null | undefined) => string[] | undefined;
  getActiveCycleId: (projectId: string | null | undefined) => string | null | undefined;
  getArchivedCycleIds: (projectId: string | null | undefined) => string[] | undefined;
  getCycleProgress: (cycleId: string | null | undefined) => CycleProgress | undefined;
  // fetch actions
  fetchCycles: (projectId: string) => Promise<CycleListItem[] | undefined>;
  fetchArchivedCycles: (projectId: string) => Promise<CycleListItem[] | undefined>;
  fetchCycle: (cycleId: string) => Promise<Cycle | null | undefined>;
  fetchProgress: (cycleId: string) => Promise<CycleProgress | undefined>;
}

/**
 * The cycles of the projects of a session (M6 design 3.1–3.8, P5a spec 2.2): each project's two lists, by its id, the
 * archived cycles and the others; each cycle's own read and its progress, by its id. A cycle's fields show as nerve
 * last answered them, of the copies the store holds (v0 design 7.7); its counts, its assignees and the counts it was
 * frozen with come from its list's row alone, which only a list gives. Every reader takes its project or cycle from
 * its caller (the route's), never from the address in the router store; the cycles of a project the project store no
 * longer gives (deleted, left, or of a workspace no longer the caller's) do not show. Its service sends with the
 * session's client; fetches do not queue. Its changes go one at a time and write nerve's answers into every copy that
 * holds the cycle (cycle-changes.ts, v0 design 7.7).
 */
export class CycleStore implements ICycleStore {
  /** Each project's cycles that are not archived, in nerve's order, reconciled between fetches and changes. */
  readonly unarchived = new ReconciledByKey<CycleListItem[]>();
  /** Each project's archived cycles, the latest archived first. */
  readonly archived = new ReconciledByKey<CycleListItem[]>();
  /** Each cycle as nerve last answered it alone; null once deleted here (a fetch out meanwhile answers null too). */
  readonly details = new ReconciledByKey<Cycle | null>();
  /** Each cycle's progress: its status, its live progress, and what it was frozen with. */
  readonly progress = new ReconciledByKey<CycleProgress>();
  // services
  private readonly service: CyclesService;
  /** The changes of the cycles, which write nerve's answers into the values above. */
  private readonly changes: CycleChanges;
  // stores
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.service = new CyclesService(api);
    this.projectOf = rootStore.projectRoot.project.getProjectById;
    this.changes = new CycleChanges(this, this.service);
  }

  /**
   * @description the cycle's fields as nerve last answered them, of its own read and its project's lists' rows; nothing
   * until one of them has it, or once the project store no longer gives its project
   */
  getCycleById = computedFn((cycleId: string | null | undefined): Cycle | undefined => {
    if (!cycleId) return undefined;
    const read = this.details.get(cycleId);
    const copies: Copy<Cycle>[] = [
      ...this.rowsOf(cycleId),
      ...(read ? [{ cycle: read, answeredAt: this.details.answeredAt(cycleId) }] : []),
    ];
    const newest = maxBy(copies, "answeredAt")?.cycle;
    return newest && this.projectOf(newest.project_id) ? newest : undefined;
  });

  /**
   * @description the cycle's row of its project's list answered last (its counts, its assignees, the counts it was
   * frozen with), with the cycle's fields as nerve last answered them (getCycleById); nothing until a list has it
   */
  getCycleRow = computedFn((cycleId: string | null | undefined): CycleListItem | undefined => {
    const cycle = this.getCycleById(cycleId);
    const row = cycle && maxBy(this.rowsOf(cycle.id), "answeredAt")?.cycle;
    return row && { ...row, ...cycle };
  });

  /** @description the project's cycles that are not archived, in nerve's order; undefined until fetched */
  getProjectCycleIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.listed(this.unarchived, projectId)?.map((row) => row.id)
  );

  /**
   * @description the project's cycle in progress: the first of its cycles that are not archived whose status, as nerve
   * last answered it, is current; null for none, undefined until fetched
   */
  getActiveCycleId = computedFn((projectId: string | null | undefined): string | null | undefined => {
    const cycleIds = this.getProjectCycleIds(projectId);
    return cycleIds && (cycleIds.find((cycleId) => this.getCycleById(cycleId)?.status === "current") ?? null);
  });

  /** @description the project's archived cycles, the latest archived first; undefined until fetched */
  getArchivedCycleIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.listed(this.archived, projectId)?.map((row) => row.id)
  );

  /** @description the cycle's progress, of a cycle the store shows (getCycleById); undefined until fetched */
  getCycleProgress = computedFn((cycleId: string | null | undefined): CycleProgress | undefined => {
    const cycle = this.getCycleById(cycleId);
    return cycle && this.progress.get(cycle.id);
  });

  /**
   * @description fetches the project's cycles that are not archived and shows them with the changes nerve confirmed
   * meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a change of session cut
   * (Reconciled.fetch)
   */
  fetchCycles = (projectId: string): Promise<CycleListItem[] | undefined> =>
    this.unarchived.fetch(projectId, () => this.service.list(projectId));

  /** @description fetches the project's archived cycles, as fetchCycles */
  fetchArchivedCycles = (projectId: string): Promise<CycleListItem[] | undefined> =>
    this.archived.fetch(projectId, () => this.service.listArchived(projectId));

  /** @description reads the cycle alone, an archived one too, as fetchCycles; null for one deleted meanwhile */
  fetchCycle = (cycleId: string): Promise<Cycle | null | undefined> =>
    this.details.fetch(cycleId, () => this.service.get(cycleId));

  /** @description reads the cycle's progress, as fetchCycles */
  fetchProgress = (cycleId: string): Promise<CycleProgress | undefined> =>
    this.progress.fetch(cycleId, () => this.service.progress(cycleId));

  // changes (cycle-changes.ts)
  create: ICycleStore["create"] = (projectId, body) => this.changes.create(projectId, body);
  update: ICycleStore["update"] = (cycleId, body) => this.changes.update(cycleId, body);
  remove: ICycleStore["remove"] = (cycleId) => this.changes.remove(cycleId);
  archive: ICycleStore["archive"] = (cycleId) => this.changes.archive(cycleId);
  unarchive: ICycleStore["unarchive"] = (cycleId) => this.changes.unarchive(cycleId);
  transfer: ICycleStore["transfer"] = (cycleId, targetCycleId) => this.changes.transfer(cycleId, targetCycleId);

  /** The project's list in lists (the unarchived or the archived), once fetched, of a project the project store gives. */
  private listed(lists: ReconciledByKey<CycleListItem[]>, projectId: string | null | undefined) {
    const project = this.projectOf(projectId);
    return project && lists.get(project.id);
  }

  /** The cycle's rows in the lists the store holds, the archived ones too, each with when nerve answered its list. */
  private rowsOf(cycleId: string): Copy<CycleListItem>[] {
    return [this.unarchived, this.archived].flatMap((lists) =>
      lists
        .values()
        .flatMap((rows) => rows.filter((row) => row.id === cycleId))
        .map((row) => ({ cycle: row, answeredAt: lists.answeredAt(row.project_id) }))
    );
  }
}
