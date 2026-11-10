/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observable, runInAction } from "mobx";
// nerve imports
import type { CycleFilters } from "@nerve/utils";

/** A project's cycles list ("default") or its archive ("archived"). */
export type CycleList = "default" | "archived";
/** What one of a project's lists is filtered and searched by. */
type ListFilters = { readonly filters: CycleFilters; readonly searchQuery: string };
type ProjectFilters = Readonly<Record<CycleList, ListFilters>>;

/** A project's lists before anything filtered or searched them. */
const UNFILTERED: ProjectFilters = {
  default: { filters: {}, searchQuery: "" },
  archived: { filters: {}, searchQuery: "" },
};

export interface ICycleFilterStore {
  // computed actions
  getFilters: (projectId: string, list: CycleList) => CycleFilters;
  getSearchQuery: (projectId: string, list: CycleList) => string;
  // actions
  /** Changes the filters given of the list, and keeps the others. */
  updateFilters: (projectId: string, list: CycleList, filters: CycleFilters) => void;
  clearAllFilters: (projectId: string, list: CycleList) => void;
  updateSearchQuery: (projectId: string, list: CycleList, query: string) => void;
}

/**
 * What each project's cycles list and archive are filtered and searched by (P5a spec 2.3), kept by the project's id
 * for the session. Every reader and change takes the project from its caller (the route's), so the store follows no
 * address (R4): it registers nothing on the RouterStore, which goes on with the next session, and leaves nothing to
 * release as its session retires. A project's search stays for the session: the address leaving the project no
 * longer clears it. The values are replaced, never changed in place: what a reader holds stays as it read it.
 */
export class CycleFilterStore implements ICycleFilterStore {
  /** Each project's lists, by its id: observed project by project. */
  private readonly projects = observable.map<string, ProjectFilters>({}, { deep: false });

  getFilters = (projectId: string, list: CycleList): CycleFilters => this.held(projectId)[list].filters;

  getSearchQuery = (projectId: string, list: CycleList): string => this.held(projectId)[list].searchQuery;

  updateFilters = (projectId: string, list: CycleList, filters: CycleFilters): void =>
    this.change(projectId, list, (held) => ({ ...held, filters: { ...held.filters, ...filters } }));

  clearAllFilters = (projectId: string, list: CycleList): void =>
    this.change(projectId, list, (held) => ({ ...held, filters: {} }));

  updateSearchQuery = (projectId: string, list: CycleList, query: string): void =>
    this.change(projectId, list, (held) => ({ ...held, searchQuery: query }));

  private held(projectId: string): ProjectFilters {
    return this.projects.get(projectId) ?? UNFILTERED;
  }

  private change(projectId: string, list: CycleList, next: (held: ListFilters) => ListFilters): void {
    runInAction(() => {
      const held = this.held(projectId);
      this.projects.set(projectId, { ...held, [list]: next(held[list]) });
    });
  }
}
