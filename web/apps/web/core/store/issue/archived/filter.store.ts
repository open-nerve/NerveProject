/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty, set } from "lodash-es";
import { action, computed, makeObservable, observable, runInAction } from "mobx";
// base class
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssueFilterType } from "@nerve/constants";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  TIssueKanbanFilters,
  IIssueFilters,
  TWorkItemFilterExpression,
  TSupportedFilterForUpdate,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { toFilterExpression, toProjectFilters } from "@/lib/issue-filters";
import type { IBaseIssueFilterStore } from "../helpers/issue-filter-helper.store";
import { IssueFilterHelperStore } from "../helpers/issue-filter-helper.store";
// types
import type { IIssueRootStore } from "../root.store";

export interface IArchivedIssuesFilter extends IBaseIssueFilterStore {
  //helper actions
  getIssueFilters(projectId: string, workspaceSlug?: string): IIssueFilters | undefined;
  // action
  fetchFilters: (workspaceSlug: string, projectId: string) => Promise<void>;
  updateFilterExpression: (
    workspaceSlug: string,
    projectId: string,
    filters: TWorkItemFilterExpression
  ) => Promise<void>;
  updateFilters: (
    workspaceSlug: string,
    projectId: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
}

export class ArchivedIssuesFilter extends IssueFilterHelperStore implements IArchivedIssuesFilter {
  // observables
  filters: { [scope: string]: IIssueFilters } = {};
  // root store
  rootIssueStore: IIssueRootStore;

  constructor(_rootStore: IIssueRootStore) {
    super();
    makeObservable(this, {
      // observables
      filters: observable,
      // computed
      issueFilters: computed,
      // actions
      fetchFilters: action,
      updateFilters: action,
    });
    // root store
    this.rootIssueStore = _rootStore;
  }

  get issueFilters() {
    const projectId = this.rootIssueStore.projectId;
    if (!projectId) return undefined;

    return this.getIssueFilters(projectId);
  }

  getIssueFilters(projectId: string, workspaceSlug = this.rootIssueStore.workspaceSlug) {
    if (!workspaceSlug) return undefined;
    const displayFilters = this.filters[JSON.stringify([workspaceSlug, projectId])] || undefined;
    if (isEmpty(displayFilters)) return undefined;

    const _filters: IIssueFilters = this.computedIssueFilters(displayFilters);

    return _filters;
  }

  fetchFilters = async (workspaceSlug: string, projectId: string) => {
    this.readLocalFilters(workspaceSlug, projectId);
  };

  private readLocalFilters(workspaceSlug: string, projectId: string) {
    this.rootIssueStore.assertActive();
    const scope = JSON.stringify([workspaceSlug, projectId]);
    if (this.filters[scope]) return;
    const _filters = this.handleIssuesLocalFilters.get(
      EIssuesStoreType.ARCHIVED,
      workspaceSlug,
      projectId,
      this.rootIssueStore.currentUserId
    );

    const richFilters = toFilterExpression(toProjectFilters(_filters?.rich_filters ?? {}));
    const displayFilters: IIssueDisplayFilterOptions = this.computedDisplayFilters({
      ..._filters?.display_filters,
      sub_issue: _filters?.display_filters?.sub_issue ?? true,
      layout: "list",
      sub_group_by: null,
    });
    const displayProperties: IIssueDisplayProperties = this.computedDisplayProperties(
      _filters?.display_properties ?? {}
    );
    const kanbanFilters: TIssueKanbanFilters = {
      group_by: [],
      sub_group_by: [],
    };
    kanbanFilters.group_by = _filters?.kanban_filters?.group_by || [];

    runInAction(() => {
      set(this.filters, [scope, "richFilters"], richFilters);
      set(this.filters, [scope, "displayFilters"], displayFilters);
      set(this.filters, [scope, "displayProperties"], displayProperties);
      set(this.filters, [scope, "kanbanFilters"], kanbanFilters);
    });
  }

  updateFilterExpression: IArchivedIssuesFilter["updateFilterExpression"] = async (
    workspaceSlug,
    projectId,
    expression
  ) => {
    this.readLocalFilters(workspaceSlug, projectId);
    const scope = JSON.stringify([workspaceSlug, projectId]);
    const filters = toFilterExpression(toProjectFilters(expression));
    runInAction(() => {
      set(this.filters, [scope, "richFilters"], filters);
    });
    this.handleIssuesLocalFilters.set(
      EIssuesStoreType.ARCHIVED,
      EIssueFilterType.FILTERS,
      workspaceSlug,
      projectId,
      this.rootIssueStore.currentUserId,
      { rich_filters: filters }
    );
    // Local persistence is complete. A subsequent list refusal belongs to the binding.
    await this.rootIssueStore.archivedIssues.fetchIssuesWithExistingPagination(workspaceSlug, projectId, "mutation");
  };

  updateFilters: IArchivedIssuesFilter["updateFilters"] = async (workspaceSlug, projectId, type, filters) => {
    this.rootIssueStore.assertActive();
    const scope = JSON.stringify([workspaceSlug, projectId]);
    const held = this.filters[scope];
    if (!held) return;
    const actor = this.rootIssueStore.currentUserId;
    switch (type) {
      case EIssueFilterType.DISPLAY_FILTERS: {
        const patch = filters as IIssueDisplayFilterOptions;
        const next = this.computedDisplayFilters({
          ...held.displayFilters,
          ...patch,
          layout: "list",
          sub_group_by: null,
        });
        runInAction(() => {
          set(this.filters, [scope, "displayFilters"], next);
        });
        this.handleIssuesLocalFilters.set(EIssuesStoreType.ARCHIVED, type, workspaceSlug, projectId, actor, {
          display_filters: next,
        });
        if (this.getShouldReFetchIssues(patch))
          await this.rootIssueStore.archivedIssues.fetchIssuesWithExistingPagination(
            workspaceSlug,
            projectId,
            "mutation"
          );
        break;
      }
      case EIssueFilterType.DISPLAY_PROPERTIES: {
        const next = { ...held.displayProperties, ...(filters as IIssueDisplayProperties) };
        runInAction(() => {
          set(this.filters, [scope, "displayProperties"], next);
        });
        this.handleIssuesLocalFilters.set(EIssuesStoreType.ARCHIVED, type, workspaceSlug, projectId, actor, {
          display_properties: next,
        });
        break;
      }
      case EIssueFilterType.KANBAN_FILTERS: {
        // List group collapse uses this shared field too; only subgroups are unsupported.
        const next = { ...held.kanbanFilters, ...(filters as TIssueKanbanFilters), sub_group_by: [] };
        runInAction(() => {
          set(this.filters, [scope, "kanbanFilters"], next);
        });
        this.handleIssuesLocalFilters.set(EIssuesStoreType.ARCHIVED, type, workspaceSlug, projectId, actor, {
          kanban_filters: next,
        });
        break;
      }
    }
  };
}
