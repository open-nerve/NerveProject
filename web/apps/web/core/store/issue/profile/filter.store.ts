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
import { toFilterExpression, toMemberFilters } from "@/lib/issue-filters";
import type { IBaseIssueFilterStore } from "../helpers/issue-filter-helper.store";
import { IssueFilterHelperStore } from "../helpers/issue-filter-helper.store";
// types
import type { IIssueRootStore } from "../root.store";

const cleanExpression = (expression: TWorkItemFilterExpression): TWorkItemFilterExpression => {
  const { project_id, ...filters } = toMemberFilters(expression);
  const clean = toFilterExpression(filters);
  if (!project_id) return clean;
  return { and: [...("and" in clean ? clean.and : []), { project_id__in: project_id.join(",") }] };
};

export interface IProfileIssuesFilter extends IBaseIssueFilterStore {
  //helper actions
  getIssueFilters(userId: string, workspaceSlug?: string): IIssueFilters | undefined;
  // action
  fetchFilters: (workspaceSlug: string, userId: string) => Promise<void>;
  updateFilterExpression: (workspaceSlug: string, userId: string, filters: TWorkItemFilterExpression) => Promise<void>;
  updateFilters: (
    workspaceSlug: string,
    projectId: string | undefined,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate,
    userId: string
  ) => Promise<void>;
}

export class ProfileIssuesFilter extends IssueFilterHelperStore implements IProfileIssuesFilter {
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
    const userId = this.rootIssueStore.userId;
    if (!userId) return undefined;

    return this.getIssueFilters(userId);
  }

  getIssueFilters = (userId: string, workspaceSlug = this.rootIssueStore.workspaceSlug) => {
    if (!workspaceSlug) return undefined;
    const displayFilters = this.filters[JSON.stringify([workspaceSlug, userId])] || undefined;
    if (isEmpty(displayFilters)) return undefined;

    const _filters: IIssueFilters = this.computedIssueFilters(displayFilters);

    return _filters;
  };

  private profileDisplayFilters(filters: IIssueDisplayFilterOptions): IIssueDisplayFilterOptions {
    const clean = this.computedDisplayFilters(filters);
    const layout = clean.layout === "kanban" ? "kanban" : "list";
    const group = clean.group_by;
    const group_by =
      group === "project" || group === "state_detail.group" || group === "priority" || group === "labels"
        ? group
        : layout === "kanban"
          ? "priority"
          : null;
    return { ...clean, layout, group_by, sub_group_by: null };
  }

  fetchFilters = async (workspaceSlug: string, userId: string) => {
    this.readLocalFilters(workspaceSlug, userId);
  };

  private readLocalFilters(workspaceSlug: string, userId: string) {
    this.rootIssueStore.assertActive();
    const scope = JSON.stringify([workspaceSlug, userId]);
    if (this.filters[scope]) return;
    const _filters = this.handleIssuesLocalFilters.get(
      EIssuesStoreType.PROFILE,
      workspaceSlug,
      userId,
      this.rootIssueStore.currentUserId
    );

    const richFilters = cleanExpression(_filters?.rich_filters ?? {});
    const displayFilters = this.profileDisplayFilters({
      ..._filters?.display_filters,
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

  updateFilterExpression: IProfileIssuesFilter["updateFilterExpression"] = async (
    workspaceSlug,
    userId,
    expression
  ) => {
    this.rootIssueStore.assertActive();
    this.readLocalFilters(workspaceSlug, userId);
    const scope = JSON.stringify([workspaceSlug, userId]);
    const filters = cleanExpression(expression);
    runInAction(() => {
      set(this.filters, [scope, "richFilters"], filters);
    });
    this.handleIssuesLocalFilters.set(
      EIssuesStoreType.PROFILE,
      EIssueFilterType.FILTERS,
      workspaceSlug,
      userId,
      this.rootIssueStore.currentUserId,
      { rich_filters: filters }
    );
    // Local persistence is complete. A subsequent list refusal belongs to the binding.
    await this.rootIssueStore.profileIssues.fetchIssuesWithExistingPagination(workspaceSlug, userId, "mutation");
  };

  updateFilters: IProfileIssuesFilter["updateFilters"] = async (workspaceSlug, _projectId, type, filters, userId) => {
    this.rootIssueStore.assertActive();
    const scope = JSON.stringify([workspaceSlug, userId]);
    const held = this.filters[scope];
    if (!held) return;
    const actor = this.rootIssueStore.currentUserId;
    switch (type) {
      case EIssueFilterType.DISPLAY_FILTERS: {
        const patch = filters as IIssueDisplayFilterOptions;
        const next = this.profileDisplayFilters({
          ...held.displayFilters,
          ...patch,
        });
        runInAction(() => {
          set(this.filters, [scope, "displayFilters"], next);
        });
        this.handleIssuesLocalFilters.set(EIssuesStoreType.PROFILE, type, workspaceSlug, userId, actor, {
          display_filters: next,
        });
        if (this.getShouldReFetchIssues(patch))
          await this.rootIssueStore.profileIssues.fetchIssuesWithExistingPagination(workspaceSlug, userId, "mutation");
        break;
      }
      case EIssueFilterType.DISPLAY_PROPERTIES: {
        const next = { ...held.displayProperties, ...(filters as IIssueDisplayProperties) };
        runInAction(() => {
          set(this.filters, [scope, "displayProperties"], next);
        });
        this.handleIssuesLocalFilters.set(EIssuesStoreType.PROFILE, type, workspaceSlug, userId, actor, {
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
        this.handleIssuesLocalFilters.set(EIssuesStoreType.PROFILE, type, workspaceSlug, userId, actor, {
          kanban_filters: next,
        });
        break;
      }
    }
  };
}
