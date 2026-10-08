/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { set } from "lodash-es";
import { action, computed, observable, makeObservable, runInAction, reaction } from "mobx";
// types
import type { TProjectDisplayFilters, TProjectFilters, TProjectAppliedDisplayFilterKeys } from "@nerve/types";
// store
import type { RootStore } from "../root.store";

export interface IProjectFilterStore {
  // observables
  searchQuery: string;
  // computed
  currentWorkspaceDisplayFilters: TProjectDisplayFilters | undefined;
  currentWorkspaceAppliedDisplayFilters: TProjectAppliedDisplayFilterKeys[] | undefined;
  currentWorkspaceFilters: TProjectFilters | undefined;
  // actions
  updateDisplayFilters: (workspaceSlug: string, displayFilters: TProjectDisplayFilters) => void;
  updateFilters: (workspaceSlug: string, filters: TProjectFilters) => void;
  updateSearchQuery: (query: string) => void;
  clearAllFilters: (workspaceSlug: string) => void;
  clearAllAppliedDisplayFilters: (workspaceSlug: string) => void;
  /** Stops following the address's workspace: the RouterStore it follows outlives the session (RootStore.dispose). */
  dispose: () => void;
}

/**
 * The projects page's filters, each workspace's by its id (v0 design 7.7): a workspace made again under a deleted
 * one's slug starts from the default. The changes take the address's slug, and find its workspace's id among the
 * caller's workspaces now.
 */
export class ProjectFilterStore implements IProjectFilterStore {
  // observables
  displayFilters: Record<string, TProjectDisplayFilters> = {};
  filters: Record<string, TProjectFilters> = {};
  searchQuery: string = "";
  // root store
  rootStore: RootStore;
  dispose: () => void;

  constructor(_rootStore: RootStore) {
    makeObservable(this, {
      // observables
      displayFilters: observable,
      filters: observable,
      searchQuery: observable.ref,
      // computed
      currentWorkspaceDisplayFilters: computed,
      currentWorkspaceAppliedDisplayFilters: computed,
      currentWorkspaceFilters: computed,
      // actions
      updateDisplayFilters: action,
      updateFilters: action,
      updateSearchQuery: action,
      clearAllFilters: action,
      clearAllAppliedDisplayFilters: action,
    });
    // root store
    this.rootStore = _rootStore;
    // initialize display filters of the current workspace
    this.dispose = reaction(
      () => this.currentWorkspaceId,
      (workspaceId) => {
        if (!workspaceId) return;
        this.initWorkspaceFilters(workspaceId);
        this.searchQuery = "";
      }
    );
  }

  /** The id of the address's workspace, when it is among the caller's. */
  private get currentWorkspaceId() {
    return this.rootStore.workspaceRoot.currentWorkspace?.id;
  }

  /** The id of the workspace the slug names, among the caller's workspaces now. */
  private idOf(workspaceSlug: string) {
    return this.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  }

  /**
   * @description get display filters of the current workspace
   */
  get currentWorkspaceDisplayFilters() {
    const workspaceId = this.currentWorkspaceId;
    if (!workspaceId) return;
    return this.displayFilters[workspaceId];
  }

  /**
   * @description get project state applied display filter of the current workspace
   * @returns {TProjectAppliedDisplayFilterKeys[] | undefined} // An array of keys of applied display filters
   */
  // TODO: Figure out a better approach for this
  get currentWorkspaceAppliedDisplayFilters() {
    const workspaceId = this.currentWorkspaceId;
    if (!workspaceId) return;
    const displayFilters = this.displayFilters[workspaceId];
    return Object.keys(displayFilters).filter(
      (key): key is TProjectAppliedDisplayFilterKeys =>
        ["my_projects", "archived_projects"].includes(key) && !!displayFilters[key as keyof TProjectDisplayFilters]
    );
  }

  /**
   * @description get filters of the current workspace
   */
  get currentWorkspaceFilters() {
    const workspaceId = this.currentWorkspaceId;
    if (!workspaceId) return;
    return this.filters[workspaceId];
  }

  /**
   * @description initialize display filters and filters of a workspace
   * @param {string} workspaceId
   */
  initWorkspaceFilters = (workspaceId: string) => {
    const displayFilters = this.displayFilters[workspaceId];
    runInAction(() => {
      this.displayFilters[workspaceId] = {
        order_by: displayFilters?.order_by || "created_at",
      };
      this.filters[workspaceId] = this.filters[workspaceId] ?? {};
    });
  };

  /**
   * @description update display filters of a workspace
   * @param {string} workspaceSlug
   * @param {TProjectDisplayFilters} displayFilters
   */
  updateDisplayFilters = (workspaceSlug: string, displayFilters: TProjectDisplayFilters) => {
    const workspaceId = this.idOf(workspaceSlug);
    if (!workspaceId) return;
    runInAction(() => {
      Object.keys(displayFilters).forEach((key) => {
        set(this.displayFilters, [workspaceId, key], displayFilters[key as keyof TProjectDisplayFilters]);
      });
    });
  };

  /**
   * @description update filters of a workspace
   * @param {string} workspaceSlug
   * @param {TProjectFilters} filters
   */
  updateFilters = (workspaceSlug: string, filters: TProjectFilters) => {
    const workspaceId = this.idOf(workspaceSlug);
    if (!workspaceId) return;
    runInAction(() => {
      Object.keys(filters).forEach((key) => {
        set(this.filters, [workspaceId, key], filters[key as keyof TProjectFilters]);
      });
    });
  };

  /**
   * @description update search query
   * @param {string} query
   */
  updateSearchQuery = (query: string) => (this.searchQuery = query);

  /**
   * @description clear all filters of a workspace
   * @param {string} workspaceSlug
   */
  clearAllFilters = (workspaceSlug: string) => {
    const workspaceId = this.idOf(workspaceSlug);
    if (!workspaceId) return;
    runInAction(() => {
      this.filters[workspaceId] = {};
    });
  };

  /**
   * @description clear project display filters of a workspace
   * @param {string} workspaceSlug
   */
  clearAllAppliedDisplayFilters = (workspaceSlug: string) => {
    const workspaceId = this.idOf(workspaceSlug);
    if (!workspaceId) return;
    runInAction(() => {
      if (!this.currentWorkspaceAppliedDisplayFilters) return;
      this.currentWorkspaceAppliedDisplayFilters.forEach((key) => {
        set(this.displayFilters, [workspaceId, key], false);
      });
    });
  };
}
