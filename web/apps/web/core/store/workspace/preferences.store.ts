/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable } from "mobx";
// nerve imports
import type { ApiClient, Workspace, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { WorkspacePreferencesService } from "@/services/workspace/workspace-preferences.service";

export interface IWorkspacePreferencesStore {
  getPreferences: (workspaceSlug: string) => WorkspacePreferences | undefined;
  fetchPreferences: (workspace: Pick<Workspace, "id" | "slug">) => Promise<WorkspacePreferences | undefined>;
  updatePreferences: (workspaceSlug: string, data: WorkspacePreferencesUpdate) => Promise<WorkspacePreferences>;
}

/**
 * The caller's own settings of the sidebar's project navigation in his workspaces (M3 design 3.18, 7.3): its
 * service sends with the session's client, which the RootStore of the session hands down. Changes go one at a time
 * (v0 design 7.7); fetches do not queue.
 */
export class WorkspacePreferencesStore implements IWorkspacePreferencesStore {
  /**
   * The caller's settings in each workspace, by the workspace's id, reconciled between their fetches and the changes
   * nerve confirmed (reconciled.ts): the settings are one document, so a change nerve confirmed wins over a fetch
   * that was out.
   */
  private readonly preferences = new ReconciledByKey<WorkspacePreferences>();
  private readonly service: WorkspacePreferencesService;
  /** The changes of the settings, sent one at a time. */
  private readonly changes = oneAtATime();

  /** workspaceOf: the caller's workspace a slug names, by his list (WorkspaceRootStore.getWorkspaceBySlug). */
  constructor(
    private readonly workspaceOf: (workspaceSlug: string) => Workspace | null,
    api: ApiClient
  ) {
    makeObservable(this, {
      fetchPreferences: action,
      updatePreferences: action,
    });
    this.service = new WorkspacePreferencesService(api);
  }

  /**
   * The caller's settings in the workspace slug names, once fetched: nothing for one that is not on his list, so
   * nothing of a workspace he left or deleted, or of one deleted whose slug names another now.
   */
  getPreferences = (workspaceSlug: string): WorkspacePreferences | undefined =>
    this.preferences.get(this.workspaceOf(workspaceSlug)?.id);

  /**
   * @description fetches the caller's settings in a workspace, nerve's defaults until he changes one, and shows them
   * with the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or
   * a change of session cut (Reconciled.fetch)
   * @returns {Promise<WorkspacePreferences | undefined>}
   */
  fetchPreferences = (workspace: Pick<Workspace, "id" | "slug">): Promise<WorkspacePreferences | undefined> =>
    this.preferences.fetch(workspace.id, () => this.service.get(workspace.slug));

  /**
   * @description changes the settings data names; the store then has nerve's answer, all of them, once it has
   * fetched them. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<WorkspacePreferences>}
   */
  updatePreferences = (workspaceSlug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> =>
    this.changes(async () => {
      const workspaceId = this.workspaceOf(workspaceSlug)?.id;
      const preferences = await this.service.update(workspaceSlug, data);
      this.preferences.confirm(workspaceId, () => preferences);
      return preferences;
    });
}
