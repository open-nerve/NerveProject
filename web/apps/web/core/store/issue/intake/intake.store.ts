/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { IntakeCounts, IntakeIssuePage, Workspace } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// store
import type { IssueRootStore } from "@/store/issue/root.store";
import type { IntakeFilters } from "./filters";
import { filterKey, listQuery } from "./filters";

/** A workspace as the store fetches its counts: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

/** A list the store fetched: the project and the workspace it is of, and the filters it names (P5a spec 2.3). */
type FetchedList = { workspaceId: string; projectId: string; filters: IntakeFilters };

/** The key of a project's list of the intake's work items some filters select: its id, and the filters' key. */
const listKey = (projectId: string, filters: IntakeFilters) => `${projectId}|${filterKey(filters)}`;

/**
 * The held page, followed by the rows of the next page it does not have yet (a work item submitted since pushes the
 * rows read down onto the next page), with the number the filters select and the next page's cursor as nerve answered
 * them (C6).
 */
function followed(held: IntakeIssuePage, next: IntakeIssuePage): IntakeIssuePage {
  const heldIds = new Set(held.data.map((row) => row.id));
  return {
    total: next.total,
    data: [...held.data, ...next.data.filter((row) => !heldIds.has(row.id))],
    next_cursor: next.next_cursor,
  };
}

export interface IIntakeStore {
  // computed actions
  pendingCount: (slug: string, projectId: string) => number | undefined;
  getList: (projectId: string, filters: IntakeFilters) => IntakeIssuePage | undefined;
  // fetch actions
  fetchCounts: (workspace: WorkspaceRef) => Promise<IntakeCounts | undefined>;
  fetchList: (
    workspace: WorkspaceRef,
    projectId: string,
    filters: IntakeFilters
  ) => Promise<IntakeIssuePage | undefined>;
  fetchMore: (projectId: string, filters: IntakeFilters) => Promise<IntakeIssuePage | undefined>;
}

/**
 * The intakes of a session's projects (M7 design 3.5, 15 A; P5a spec 2.3): a child of the session's issue root store,
 * through which it reaches its service (intakeApi) and its siblings. It holds each workspace's pending counts, by its
 * id, as nerve last answered them; only nerve counts them (by status, a snooze come due, what a guest sees), so the
 * store never does. Its readers name the workspace by the address's slug, as the caller's workspaces list it now,
 * never by the router store: a workspace no longer his gives nothing (v0 design 7.7). It holds each project's lists of
 * the work items some filters select, by the project's id and the filters' key, the pages read so far as one page: its
 * own copy of nerve's rows, which a fetch of the first page replaces. Fetches do not queue.
 */
export class IntakeStore implements IIntakeStore {
  /** Each workspace's pending counts, by its id, reconciled between fetches (reconciled.ts). */
  private readonly counts = new ReconciledByKey<IntakeCounts>();
  /**
   * Each project's list of the intake's work items some filters select, by listKey: the pages read so far as one page,
   * with the number the filters select and the next page's cursor, reconciled between its fetches and the next pages'
   * reads (reconciled.ts).
   */
  private readonly lists = new ReconciledByKey<IntakeIssuePage>();
  /** The lists fetched, by their keys: whose project and workspace each is, and the filters it names. */
  private readonly fetched = new Map<string, FetchedList>();

  constructor(private readonly root: IssueRootStore) {}

  /**
   * @description the pending work items of the project's intake in the caller's workspace slug names, as nerve last
   * counted them; undefined until fetched, for a project nerve did not count (its intake off, archived, not the
   * caller's), or for a slug that names no workspace of his now
   */
  pendingCount = computedFn((slug: string, projectId: string): number | undefined => {
    const workspace = this.root.rootStore.workspaceRoot.getWorkspaceBySlug(slug);
    if (!workspace) return undefined;
    return this.counts.get(workspace.id)?.data.find((count) => count.project_id === projectId)?.pending;
  });

  /**
   * @description fetches the workspace's pending counts; gives them, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch); a refusal fails it, the counts as they were
   */
  fetchCounts = (workspace: WorkspaceRef): Promise<IntakeCounts | undefined> =>
    this.counts.fetch(workspace.id, () => this.root.intakeApi.counts(workspace.slug));

  /**
   * @description the work items of the project's intake the filters select, the pages read so far as one, with the
   * number the filters select and the next page's cursor (none past the last); undefined until fetched
   */
  getList = computedFn((projectId: string, filters: IntakeFilters): IntakeIssuePage | undefined =>
    this.lists.get(listKey(projectId, filters))
  );

  /**
   * @description fetches the first page of the project's intake the filters select, in place of every page held of
   * them (a refresh): a next page's read out then, or sent before it answers, adds nothing (fetchMore). Gives the page,
   * or undefined as fetchCounts; a refusal fails it, the list as it was
   */
  fetchList = (
    workspace: WorkspaceRef,
    projectId: string,
    filters: IntakeFilters
  ): Promise<IntakeIssuePage | undefined> => {
    const list = listKey(projectId, filters);
    this.fetched.set(list, { workspaceId: workspace.id, projectId, filters });
    return this.lists.fetch(list, () => this.root.intakeApi.list(projectId, listQuery(filters)));
  };

  /**
   * @description reads the page after the held page of the project's intake the filters select, by the cursor nerve
   * answered (C6), and adds it (followed) if the held page still ends there as it answers: a refresh answered
   * meanwhile ends elsewhere, and the page read is not the one after it (P4 whole-branch I1). Asks nerve nothing past
   * the last page or for a list not fetched; gives the list, or undefined as fetchList
   */
  fetchMore = (projectId: string, filters: IntakeFilters): Promise<IntakeIssuePage | undefined> => {
    const list = listKey(projectId, filters);
    const cursor = this.lists.get(list)?.next_cursor;
    if (!cursor) return Promise.resolve(undefined);
    const read = () => this.root.intakeApi.list(projectId, listQuery(filters, cursor));
    const merge = (held: IntakeIssuePage, next: IntakeIssuePage) =>
      held.next_cursor === cursor ? followed(held, next) : held;
    return this.lists.fetchPart(list, cursor, read, merge);
  };
}
