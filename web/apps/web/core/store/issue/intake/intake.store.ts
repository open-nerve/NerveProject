/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { IntakeCounts, Workspace } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// store
import type { IssueRootStore } from "@/store/issue/root.store";

/** A workspace as the store fetches its counts: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface IIntakeStore {
  // computed actions
  pendingCount: (slug: string, projectId: string) => number | undefined;
  // fetch actions
  fetchCounts: (workspace: WorkspaceRef) => Promise<IntakeCounts | undefined>;
}

/**
 * The intakes of a session's projects (M7 design 3.5, 15 A; P5a spec 2.3): a child of the session's issue root store,
 * through which it reaches its service (intakeApi) and its siblings. It holds each workspace's pending counts, by its
 * id, as nerve last answered them; only nerve counts them (by status, a snooze come due, what a guest sees), so the
 * store never does. Its readers name the workspace by the address's slug, as the caller's workspaces list it now,
 * never by the router store: a workspace no longer his gives nothing (v0 design 7.7). Fetches do not queue.
 */
export class IntakeStore implements IIntakeStore {
  /** Each workspace's pending counts, by its id, reconciled between fetches (reconciled.ts). */
  private readonly counts = new ReconciledByKey<IntakeCounts>();

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
}
