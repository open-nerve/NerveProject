/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspacePreferencesService } from "@/services/workspace/workspace-preferences.service";

export interface IWorkspacePreferencesStore {
  getPreferences: (workspaceSlug: string) => WorkspacePreferences | undefined;
  fetchPreferences: (workspaceSlug: string) => Promise<WorkspacePreferences | undefined>;
  updatePreferences: (workspaceSlug: string, data: WorkspacePreferencesUpdate) => Promise<WorkspacePreferences>;
}

/**
 * The caller's own settings of the sidebar's project navigation in his workspaces (M3 design 3.18, 7.3): its
 * service sends with the session's client, which the RootStore of the session hands down. Changes go one at a time
 * (v0 design 7.7); fetches do not queue.
 */
export class WorkspacePreferencesStore implements IWorkspacePreferencesStore {
  /** The caller's settings of the sidebar's project navigation, by workspace slug, as nerve last gave them. */
  preferencesMap: Record<string, WorkspacePreferences> = {};
  private readonly service: WorkspacePreferencesService;
  /** The changes of the settings, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(api: ApiClient) {
    makeObservable(this, {
      preferencesMap: observable,
      fetchPreferences: action,
      updatePreferences: action,
    });
    this.service = new WorkspacePreferencesService(api);
  }

  /** The caller's settings in the workspace, once fetched. */
  getPreferences = (workspaceSlug: string): WorkspacePreferences | undefined => this.preferencesMap[workspaceSlug];

  /**
   * @description fetches the caller's settings in a workspace, nerve's defaults until he changes one, and gives
   * them. A change of session while they load is no failure: the new session's store fetches its own
   * (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspacePreferences | undefined>}
   */
  fetchPreferences = async (workspaceSlug: string): Promise<WorkspacePreferences | undefined> => {
    try {
      const preferences = await this.service.get(workspaceSlug);
      runInAction(() => {
        this.preferencesMap[workspaceSlug] = preferences;
      });
      return preferences;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description changes the settings data names; the store then has nerve's answer, all of them. Fails, changing
   * nothing, when nerve refuses.
   * @returns {Promise<WorkspacePreferences>}
   */
  updatePreferences = (workspaceSlug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> =>
    this.changes(async () => {
      const preferences = await this.service.update(workspaceSlug, data);
      runInAction(() => {
        this.preferencesMap[workspaceSlug] = preferences;
      });
      return preferences;
    });
}
