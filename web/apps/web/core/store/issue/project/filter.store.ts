/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty, set } from "lodash-es";
import type { ApiClient, ProjectPreferences, ProjectDisplayFilters } from "@nerve/api-client";
import { oneAtATime } from "@/lib/one-at-a-time";
import { projectListQuery, toFilterExpression, toProjectFilters } from "@/lib/issue-filters";
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";
import { action, computed, makeObservable, observable, runInAction } from "mobx";
// base class
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssueFilterType } from "@nerve/constants";
import type {
  IIssueDisplayFilterOptions,
  TIssueKanbanFilters,
  IIssueFilters,
  TWorkItemFilterExpression,
  TSupportedFilterForUpdate,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import type { IBaseIssueFilterStore } from "../helpers/issue-filter-helper.store";
import { IssueFilterHelperStore } from "../helpers/issue-filter-helper.store";
// types
import type { IIssueRootStore } from "../root.store";

const groupWire: Record<string, string> = {
  state: "state_id",
  "state_detail.group": "state_group",
  priority: "priority",
  labels: "label_id",
  assignees: "assignee_id",
  created_by: "created_by_id",
  target_date: "target_date",
};
const orderWire: Record<string, string> = {
  state__name: "state_name",
  labels__name: "label_name",
  assignees__first_name: "assignee_name",
};
function controlOrder(value: string): string {
  const minus = value.startsWith("-") ? "-" : "";
  const bare = minus ? value.slice(1) : value;
  return minus + (Object.entries(orderWire).find(([, wire]) => wire === bare)?.[0] ?? bare);
}

export interface IProjectIssuesFilter extends IBaseIssueFilterStore {
  //helper actions
  getIssueFilters(projectId: string): IIssueFilters | undefined;
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

export class ProjectIssuesFilter extends IssueFilterHelperStore implements IProjectIssuesFilter {
  // observables
  filters: { [projectId: string]: IIssueFilters } = {};
  // root store
  rootIssueStore: IIssueRootStore;
  // services
  private readonly service: ProjectPreferencesService;
  private readonly values = new Map<string, ProjectPreferences>();
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();
  private readonly versions = new Map<string, number>();
  private readonly reads = new Map<string, number>();

  constructor(_rootStore: IIssueRootStore, api: ApiClient) {
    super();
    makeObservable(this, {
      // observables
      filters: observable,
      // computed
      issueFilters: computed,
      // actions
      fetchFilters: action,
      updateFilterExpression: action,
      updateFilters: action,
    });
    // root store
    this.rootIssueStore = _rootStore;
    // services
    this.service = new ProjectPreferencesService(api);
  }

  get issueFilters() {
    const projectId = this.rootIssueStore.projectId;
    if (!projectId) return undefined;

    return this.getIssueFilters(projectId);
  }

  getIssueFilters(projectId: string) {
    const displayFilters = this.filters[projectId] || undefined;
    if (isEmpty(displayFilters)) return undefined;

    return this.computedIssueFilters(displayFilters);
  }

  private show(projectId: string, value: ProjectPreferences) {
    const groupValues: NonNullable<IIssueDisplayFilterOptions["group_by"]>[] = [
      "state",
      "priority",
      "labels",
      "created_by",
      "state_detail.group",
      "project",
      "assignees",
      "target_date",
    ];
    const orderValues: NonNullable<IIssueDisplayFilterOptions["order_by"]>[] = [
      "-created_at",
      "created_at",
      "updated_at",
      "-updated_at",
      "priority",
      "-priority",
      "sort_order",
      "state__name",
      "-state__name",
      "assignees__first_name",
      "-assignees__first_name",
      "labels__name",
      "-labels__name",
      "target_date",
      "-target_date",
      "start_date",
      "-start_date",
      "link_count",
      "-link_count",
      "sub_issues_count",
      "-sub_issues_count",
    ];
    const d = value.display_filters;
    const display: IIssueDisplayFilterOptions = {
      ...d,
      group_by: groupValues.find((v) => groupWire[v] === d.group_by) ?? null,
      sub_group_by: groupValues.find((v) => groupWire[v] === d.sub_group_by) ?? null,
      order_by: orderValues.find((v) => v === controlOrder(d.order_by)) ?? "-created_at",
      layout: ["list", "kanban", "calendar", "spreadsheet"].includes(d.layout) ? d.layout : "list",
      calendar: { ...d.calendar, layout: d.calendar.layout === "week" ? "week" : "month" },
    };
    runInAction(() => {
      set(this.filters, [projectId, "richFilters"], toFilterExpression(value.filters));
      set(this.filters, [projectId, "displayFilters"], display);
      set(this.filters, [projectId, "displayProperties"], value.display_properties);
    });
  }

  fetchFilters = async (workspaceSlug: string, projectId: string) => {
    this.rootIssueStore.assertActive();
    const sequence = (this.reads.get(projectId) ?? 0) + 1;
    this.reads.set(projectId, sequence);
    const versions = ["filters", "display_filters", "display_properties"].map(
      (field) => this.versions.get(`${projectId}:${field}`) ?? 0
    );
    const value = await this.service.get(projectId);
    this.rootIssueStore.assertActive();
    if (this.reads.get(projectId) !== sequence) return;
    const held = this.values.get(projectId);
    if (held) {
      if ((this.versions.get(`${projectId}:filters`) ?? 0) !== versions[0]) value.filters = held.filters;
      if ((this.versions.get(`${projectId}:display_filters`) ?? 0) !== versions[1])
        value.display_filters = held.display_filters;
      if ((this.versions.get(`${projectId}:display_properties`) ?? 0) !== versions[2])
        value.display_properties = held.display_properties;
    }
    this.values.set(projectId, value);
    this.show(projectId, value);
    // The server owns three preference fields; group collapse is already saved in this session.
    if (this.filters[projectId]?.kanbanFilters) return;
    const currentUserId = this.rootIssueStore.currentUserId;
    const local = currentUserId
      ? this.handleIssuesLocalFilters.get(EIssuesStoreType.PROJECT, workspaceSlug, projectId, currentUserId)
      : undefined;
    runInAction(() =>
      set(this.filters, [projectId, "kanbanFilters"], local?.kanban_filters ?? { group_by: [], sub_group_by: [] })
    );
  };

  private change<K extends "filters" | "display_filters" | "display_properties">(
    projectId: string,
    field: K,
    make: (held: ProjectPreferences[K]) => ProjectPreferences[K]
  ): Promise<void> {
    const key = `${projectId}:${field}`;
    let queue = this.queues.get(key);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(key, queue);
    }
    return queue(async () => {
      this.rootIssueStore.assertActive();
      const held = this.values.get(projectId);
      if (!held) throw new Error("Project preferences not loaded");
      const answer = await this.service.update(projectId, { [field]: make(held[field]) });
      this.rootIssueStore.assertActive();
      const current = this.values.get(projectId);
      if (!current) throw new Error("Project preferences not loaded");
      const confirmed = { ...current, [field]: answer[field] };
      this.values.set(projectId, confirmed);
      this.versions.set(key, (this.versions.get(key) ?? 0) + 1);
      this.show(projectId, confirmed);
    });
  }

  /**
   * NOTE: This method is designed as a fallback function for the work item filter store.
   * Only use this method directly when initializing filter instances.
   * For regular filter updates, use this method as a fallback function for the work item filter store methods instead.
   */
  updateFilterExpression: IProjectIssuesFilter["updateFilterExpression"] = (_workspaceSlug, projectId, filters) =>
    this.change(projectId, "filters", () => toProjectFilters(filters));

  updateFilters: IProjectIssuesFilter["updateFilters"] = async (workspaceSlug, projectId, type, filters) => {
    switch (type) {
      case EIssueFilterType.DISPLAY_FILTERS:
        return this.change(projectId, "display_filters", (held) => {
          const update = filters as IIssueDisplayFilterOptions;
          const result: ProjectDisplayFilters = {
            ...held,
            ...update,
            calendar: { ...held.calendar, ...update.calendar },
          };
          if (update.group_by !== undefined)
            result.group_by = update.group_by ? (groupWire[update.group_by] ?? null) : null;
          if (update.sub_group_by !== undefined)
            result.sub_group_by = update.sub_group_by ? (groupWire[update.sub_group_by] ?? null) : null;
          if (update.order_by !== undefined)
            result.order_by = projectListQuery({}, { order_by: update.order_by }).order_by ?? "-created_at";
          if (result.group_by === null) result.sub_group_by = null;
          if (result.layout === "kanban" && result.group_by === result.sub_group_by) result.sub_group_by = null;
          if (result.layout === "kanban" && result.group_by === null) result.group_by = "state_id";
          return result;
        });
      case EIssueFilterType.DISPLAY_PROPERTIES:
        return this.change(projectId, "display_properties", (held) => ({ ...held, ...filters }));
      case EIssueFilterType.KANBAN_FILTERS: {
        const value = { ...this.filters[projectId]?.kanbanFilters, ...filters } as TIssueKanbanFilters;
        const userId = this.rootIssueStore.currentUserId;
        if (userId)
          this.handleIssuesLocalFilters.set(EIssuesStoreType.PROJECT, type, workspaceSlug, projectId, userId, {
            kanban_filters: value,
          });
        runInAction(() => set(this.filters, [projectId, "kanbanFilters"], value));
      }
    }
  };
}
