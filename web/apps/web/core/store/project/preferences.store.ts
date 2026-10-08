/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable } from "mobx";
// nerve imports
import type { ApiClient, Project, ProjectNavigation } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";

/** A change of a tab bar: what it makes of the tab bar it is made to. */
type NavigationChange = (held: ProjectNavigation) => ProjectNavigation;

export interface IProjectPreferencesStore {
  getNavigation: (projectId: string) => ProjectNavigation | undefined;
  fetchNavigation: (projectId: string) => Promise<ProjectNavigation | undefined>;
  updateNavigation: (projectId: string, change: NavigationChange) => Promise<ProjectNavigation>;
}

/** nerve's tab bar of one who has changed none (M3 design 3.18): the project opens on its work items, none hidden. */
const DEFAULT_NAVIGATION: ProjectNavigation = { default_tab: "work_items", hide_in_more_menu: [] };

/**
 * The tab bar of each project's header as the caller has it, the navigation of his settings in the project (M3 design
 * 3.18, 7.3). Their other field, the project's place in his sidebar, nerve also gives with the project
 * (Project.sort_order): the project store holds and changes it. The service sends with the session's client, which
 * the RootStore of the session hands down; changes go one at a time and the store writes nerve's answers (v0 design
 * 7.7); fetches do not queue.
 */
export class ProjectPreferencesStore implements IProjectPreferencesStore {
  /**
   * The tab bar in each project, by the project's id, reconciled between its fetches and the changes nerve confirmed
   * (reconciled.ts): it is one document, so a change nerve confirmed wins over a fetch that was out.
   */
  private readonly navigation = new ReconciledByKey<ProjectNavigation>();
  private readonly service: ProjectPreferencesService;
  /** The changes of the tab bars, sent one at a time. */
  private readonly changes = oneAtATime();

  /** projectOf: the project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  constructor(
    private readonly projectOf: (projectId: string) => Project | undefined,
    api: ApiClient
  ) {
    makeObservable(this, {
      fetchNavigation: action,
      updateNavigation: action,
    });
    this.service = new ProjectPreferencesService(api);
  }

  /**
   * The caller's tab bar in the project, once fetched: nothing of a project the project store no longer gives, one
   * deleted or left, or of a workspace no longer his.
   */
  getNavigation = (projectId: string): ProjectNavigation | undefined =>
    this.projectOf(projectId) ? this.navigation.get(projectId) : undefined;

  /**
   * @description fetches the caller's tab bar in a project, nerve's default until he changes it, and shows it with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchNavigation = (projectId: string): Promise<ProjectNavigation | undefined> =>
    this.navigation.fetch(projectId, async () => (await this.service.get(projectId)).navigation);

  /**
   * @description changes the caller's tab bar in a project, which nerve replaces whole: change is made, in the
   * change's turn, to the tab bar nerve last answered (its default until fetched), so that a change asked for before
   * the one before it is answered keeps that one; the store then has nerve's answer. Fails, changing nothing, when
   * nerve refuses.
   */
  updateNavigation = (projectId: string, change: NavigationChange): Promise<ProjectNavigation> =>
    this.changes(async () => {
      const held = this.getNavigation(projectId) ?? DEFAULT_NAVIGATION;
      const preferences = await this.service.update(projectId, { navigation: change(held) });
      this.navigation.confirm(projectId, () => preferences.navigation);
      return preferences.navigation;
    });
}
