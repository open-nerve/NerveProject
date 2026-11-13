/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observable, runInAction } from "mobx";
// nerve imports
import type { ModuleFilters } from "@nerve/utils";

/** A project's modules list ("default") or its archive ("archived"). */
export type ModuleListKind = "default" | "archived";
/** What one of a project's lists is filtered and searched by. */
type ListFilters = { readonly filters: ModuleFilters; readonly searchQuery: string };
type ProjectFilters = Readonly<Record<ModuleListKind, ListFilters>>;

/** A project's lists before anything filtered or searched them. */
const UNFILTERED: ProjectFilters = {
  default: { filters: {}, searchQuery: "" },
  archived: { filters: {}, searchQuery: "" },
};

export interface IModuleFilterStore {
  // computed actions
  getFilters: (projectId: string, list: ModuleListKind) => ModuleFilters;
  getSearchQuery: (projectId: string, list: ModuleListKind) => string;
  // actions
  /** Changes the filters given of the list, and keeps the others. */
  updateFilters: (projectId: string, list: ModuleListKind, filters: ModuleFilters) => void;
  clearAllFilters: (projectId: string, list: ModuleListKind) => void;
  updateSearchQuery: (projectId: string, list: ModuleListKind, query: string) => void;
}

/**
 * What each project's modules list and archive are filtered and searched by (P6a spec 2.4), kept by the project's id
 * for the session. Every reader and change takes the project from its caller (the route's), so the store follows no
 * address (R4): it registers nothing on the RouterStore, which goes on with the next session, and leaves nothing to
 * release as its session retires. The filters name members, so they stay in the session: never kept in the browser,
 * where they would reach the next account to sign in there. How the lists are laid out and ordered is the browser's
 * (the list's display settings), not this store's. The values are replaced, never changed in place: what a reader
 * holds stays as it read it.
 */
export class ModuleFilterStore implements IModuleFilterStore {
  /** Each project's lists, by its id: observed project by project. */
  private readonly projects = observable.map<string, ProjectFilters>({}, { deep: false });

  getFilters = (projectId: string, list: ModuleListKind): ModuleFilters => this.held(projectId)[list].filters;

  getSearchQuery = (projectId: string, list: ModuleListKind): string => this.held(projectId)[list].searchQuery;

  updateFilters = (projectId: string, list: ModuleListKind, filters: ModuleFilters): void =>
    this.change(projectId, list, (held) => ({ ...held, filters: { ...held.filters, ...filters } }));

  clearAllFilters = (projectId: string, list: ModuleListKind): void =>
    this.change(projectId, list, (held) => ({ ...held, filters: {} }));

  updateSearchQuery = (projectId: string, list: ModuleListKind, query: string): void =>
    this.change(projectId, list, (held) => ({ ...held, searchQuery: query }));

  private held(projectId: string): ProjectFilters {
    return this.projects.get(projectId) ?? UNFILTERED;
  }

  private change(projectId: string, list: ModuleListKind, next: (held: ListFilters) => ListFilters): void {
    runInAction(() => {
      const held = this.held(projectId);
      this.projects.set(projectId, { ...held, [list]: next(held[list]) });
    });
  }
}
