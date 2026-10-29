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
  getIssueFilters(userId: string): IIssueFilters | undefined;
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
  filters: { [userId: string]: IIssueFilters } = {};
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

  getIssueFilters(userId: string) {
    const displayFilters = this.filters[userId] || undefined;
    if (isEmpty(displayFilters)) return undefined;

    const _filters: IIssueFilters = this.computedIssueFilters(displayFilters);

    return _filters;
  }

  fetchFilters = async (workspaceSlug: string, userId: string) => {
    const _filters = this.handleIssuesLocalFilters.get(
      EIssuesStoreType.PROFILE,
      workspaceSlug,
      userId,
      this.rootIssueStore.currentUserId
    );

    const richFilters = cleanExpression(_filters?.rich_filters ?? {});
    const displayFilters: IIssueDisplayFilterOptions = this.computedDisplayFilters(_filters?.display_filters ?? {});
    const displayProperties: IIssueDisplayProperties = this.computedDisplayProperties(
      _filters?.display_properties ?? {}
    );
    const kanbanFilters: TIssueKanbanFilters = {
      group_by: _filters?.kanban_filters?.group_by || [],
      sub_group_by: _filters?.kanban_filters?.sub_group_by || [],
    };

    runInAction(() => {
      set(this.filters, [userId, "richFilters"], richFilters);
      set(this.filters, [userId, "displayFilters"], displayFilters);
      set(this.filters, [userId, "displayProperties"], displayProperties);
      set(this.filters, [userId, "kanbanFilters"], kanbanFilters);
    });
  };

  /**
   * NOTE: This method is designed as a fallback function for the work item filter store.
   * Only use this method directly when initializing filter instances.
   * For regular filter updates, use this method as a fallback function for the work item filter store methods instead.
   */
  updateFilterExpression: IProfileIssuesFilter["updateFilterExpression"] = async (workspaceSlug, userId, filters) => {
    try {
      filters = cleanExpression(filters);
      runInAction(() => {
        set(this.filters, [userId, "richFilters"], filters);
      });

      this.rootIssueStore.profileIssues.fetchIssuesWithExistingPagination(workspaceSlug, userId, "mutation");
      this.handleIssuesLocalFilters.set(
        EIssuesStoreType.PROFILE,
        EIssueFilterType.FILTERS,
        workspaceSlug,
        userId,
        this.rootIssueStore.currentUserId,
        {
          rich_filters: filters,
        }
      );
    } catch (error) {
      console.log("error while updating rich filters", error);
      throw error;
    }
  };

  updateFilters: IProfileIssuesFilter["updateFilters"] = async (workspaceSlug, _projectId, type, filters, userId) => {
    try {
      if (isEmpty(this.filters) || isEmpty(this.filters[userId])) return;

      const _filters = {
        richFilters: this.filters[userId].richFilters,
        displayFilters: this.filters[userId].displayFilters as IIssueDisplayFilterOptions,
        displayProperties: this.filters[userId].displayProperties as IIssueDisplayProperties,
        kanbanFilters: this.filters[userId].kanbanFilters as TIssueKanbanFilters,
      };

      switch (type) {
        case EIssueFilterType.DISPLAY_FILTERS: {
          const updatedDisplayFilters = filters as IIssueDisplayFilterOptions;
          _filters.displayFilters = { ..._filters.displayFilters, ...updatedDisplayFilters };

          // set sub_group_by to null if group_by is set to null
          if (_filters.displayFilters.group_by === null) {
            _filters.displayFilters.sub_group_by = null;
            updatedDisplayFilters.sub_group_by = null;
          }
          // set sub_group_by to null if layout is switched to kanban group_by and sub_group_by are same
          if (
            _filters.displayFilters.layout === "kanban" &&
            _filters.displayFilters.group_by === _filters.displayFilters.sub_group_by
          ) {
            _filters.displayFilters.sub_group_by = null;
            updatedDisplayFilters.sub_group_by = null;
          }
          // set group_by to priority if layout is switched to kanban and group_by is null
          if (_filters.displayFilters.layout === "kanban" && _filters.displayFilters.group_by === null) {
            _filters.displayFilters.group_by = "priority";
            updatedDisplayFilters.group_by = "priority";
          }

          runInAction(() => {
            Object.keys(updatedDisplayFilters).forEach((_key) => {
              set(
                this.filters,
                [userId, "displayFilters", _key],
                updatedDisplayFilters[_key as keyof IIssueDisplayFilterOptions]
              );
            });
          });

          this.rootIssueStore.profileIssues.fetchIssuesWithExistingPagination(workspaceSlug, userId, "mutation");

          this.handleIssuesLocalFilters.set(
            EIssuesStoreType.PROFILE,
            type,
            workspaceSlug,
            userId,
            this.rootIssueStore.currentUserId,
            {
              display_filters: _filters.displayFilters,
            }
          );

          break;
        }
        case EIssueFilterType.DISPLAY_PROPERTIES: {
          const updatedDisplayProperties = filters as IIssueDisplayProperties;
          _filters.displayProperties = { ..._filters.displayProperties, ...updatedDisplayProperties };

          runInAction(() => {
            Object.keys(updatedDisplayProperties).forEach((_key) => {
              set(
                this.filters,
                [userId, "displayProperties", _key],
                updatedDisplayProperties[_key as keyof IIssueDisplayProperties]
              );
            });
          });

          this.handleIssuesLocalFilters.set(
            EIssuesStoreType.PROFILE,
            type,
            workspaceSlug,
            userId,
            this.rootIssueStore.currentUserId,
            {
              display_properties: _filters.displayProperties,
            }
          );
          break;
        }

        case EIssueFilterType.KANBAN_FILTERS: {
          const updatedKanbanFilters = filters as TIssueKanbanFilters;
          _filters.kanbanFilters = { ..._filters.kanbanFilters, ...updatedKanbanFilters };

          const currentUserId = this.rootIssueStore.currentUserId;
          if (currentUserId)
            this.handleIssuesLocalFilters.set(
              EIssuesStoreType.PROFILE,
              type,
              workspaceSlug,
              userId,
              this.rootIssueStore.currentUserId,
              {
                kanban_filters: _filters.kanbanFilters,
              }
            );

          runInAction(() => {
            Object.keys(updatedKanbanFilters).forEach((_key) => {
              set(
                this.filters,
                [userId, "kanbanFilters", _key],
                updatedKanbanFilters[_key as keyof TIssueKanbanFilters]
              );
            });
          });

          break;
        }
        default:
          break;
      }
    } catch (error) {
      if (userId) this.fetchFilters(workspaceSlug, userId);
      throw error;
    }
  };
}
