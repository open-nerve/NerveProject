/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { isEmpty, set } from "lodash-es";
import { action, makeObservable, observable, runInAction } from "mobx";
import type { IssueListDisplayFilters, ProjectPreferences } from "@nerve/api-client";
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssueFilterType } from "@nerve/constants";
import type {
  EIssuesStoreType,
  IIssueDisplayFilterOptions,
  IIssueFilters,
  TIssueKanbanFilters,
  TSupportedFilterForUpdate,
  TWorkItemFilterExpression,
} from "@nerve/types";
import { oneAtATime } from "@/lib/one-at-a-time";
import { projectListQuery, toFilterExpression, toProjectFilters } from "@/lib/issue-filters";
import type { IIssueRootStore } from "../root.store";
import { IssueFilterHelperStore } from "./issue-filter-helper.store";

/** The fields every list page's own preferences hold on nerve: the project's page and a cycle's page alike. */
type PagePreferenceField = "filters" | "display_filters" | "display_properties";
export type PagePreferences = Pick<ProjectPreferences, PagePreferenceField>;

/** A page's preferences on nerve, by the page's id; each field named in an update replaces that field whole. */
export interface PagePreferencesService {
  get(id: string): Promise<PagePreferences>;
  update(id: string, body: Partial<PagePreferences>): Promise<PagePreferences>;
}

export interface IPagePreferencesFilter {
  filters: Record<string, IIssueFilters>;
  getIssueFilters(id: string): IIssueFilters | undefined;
  fetchFilters: (workspaceSlug: string, id: string) => Promise<void>;
  updateFilterExpression: (workspaceSlug: string, id: string, filters: TWorkItemFilterExpression) => Promise<void>;
  updateFilters: (
    workspaceSlug: string,
    id: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
}

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
  "attachment_count",
  "-attachment_count",
  "sub_issues_count",
  "-sub_issues_count",
];
const fields: PagePreferenceField[] = ["filters", "display_filters", "display_properties"];

function controlOrder(value: string): string {
  const minus = value.startsWith("-") ? "-" : "";
  const bare = minus ? value.slice(1) : value;
  return minus + (Object.entries(orderWire).find(([, wire]) => wire === bare)?.[0] ?? bare);
}

/**
 * One list page's preferences per page id (v0 design 7.7): reads in order, one queue of whole-field changes per field,
 * nerve's answers shown in the controls' vocabulary, and the kanban groups' collapse kept locally under the page's
 * store type. A page gives its service and store type and passes its id on each call; the base reads no route.
 */
export class PagePreferencesFilter extends IssueFilterHelperStore implements IPagePreferencesFilter {
  filters: Record<string, IIssueFilters> = {};
  private readonly values = new Map<string, PagePreferences>();
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();
  private readonly versions = new Map<string, number>();
  private readonly reads = new Map<string, number>();

  constructor(
    protected readonly rootIssueStore: IIssueRootStore,
    private readonly service: PagePreferencesService,
    private readonly storeType: EIssuesStoreType
  ) {
    super();
    makeObservable(this, {
      filters: observable,
      fetchFilters: action,
      updateFilterExpression: action,
      updateFilters: action,
    });
  }

  getIssueFilters(id: string) {
    const displayFilters = this.filters[id] || undefined;
    if (isEmpty(displayFilters)) return undefined;

    return this.computedIssueFilters(displayFilters);
  }

  private show(id: string, value: PagePreferences) {
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
      set(this.filters, [id, "richFilters"], toFilterExpression(value.filters));
      set(this.filters, [id, "displayFilters"], display);
      set(this.filters, [id, "displayProperties"], value.display_properties);
    });
  }

  private version(id: string, field: PagePreferenceField): number {
    return this.versions.get(`${id}:${field}`) ?? 0;
  }

  fetchFilters = async (workspaceSlug: string, id: string) => {
    this.rootIssueStore.assertActive();
    const sequence = (this.reads.get(id) ?? 0) + 1;
    this.reads.set(id, sequence);
    const versions = fields.map((field) => this.version(id, field));
    const value = await this.service.get(id);
    this.rootIssueStore.assertActive();
    if (this.reads.get(id) !== sequence) return;
    // A field confirmed after this read started is newer than what the read returned.
    const held = this.values.get(id);
    if (held)
      fields.forEach((field, at) => {
        if (this.version(id, field) !== versions[at]) keep(held, value, field);
      });
    this.values.set(id, value);
    this.show(id, value);
    // The server owns three preference fields; group collapse is already saved in this session.
    if (this.filters[id]?.kanbanFilters) return;
    const currentUserId = this.rootIssueStore.currentUserId;
    const local = currentUserId
      ? this.handleIssuesLocalFilters.get(this.storeType, workspaceSlug, id, currentUserId)
      : undefined;
    runInAction(() =>
      set(this.filters, [id, "kanbanFilters"], local?.kanban_filters ?? { group_by: [], sub_group_by: [] })
    );
  };

  /**
   * Replaces one field whole with what `next` gives once the field's earlier changes settled. It needs nothing read
   * (P5b spec 2.4): without a read the answer, the page's whole preferences, seeds the page.
   */
  private replace<K extends PagePreferenceField>(id: string, field: K, next: () => PagePreferences[K]): Promise<void> {
    const key = `${id}:${field}`;
    let queue = this.queues.get(key);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(key, queue);
    }
    return queue(async () => {
      this.rootIssueStore.assertActive();
      const answer = await this.service.update(id, { [field]: next() });
      this.rootIssueStore.assertActive();
      const current = this.values.get(id);
      const confirmed = current ? { ...current, [field]: answer[field] } : answer;
      this.values.set(id, confirmed);
      this.versions.set(key, this.version(id, field) + 1);
      this.show(id, confirmed);
    });
  }

  /** Changes part of one field, from its last confirmed value; that needs the page's preferences read. */
  private merge<K extends PagePreferenceField>(
    id: string,
    field: K,
    make: (held: PagePreferences[K]) => PagePreferences[K]
  ): Promise<void> {
    return this.replace(id, field, () => {
      const held = this.values.get(id);
      if (!held) throw new Error("Page preferences not loaded");
      return make(held[field]);
    });
  }

  /**
   * NOTE: This method is designed as a fallback function for the work item filter store.
   * Only use this method directly when initializing filter instances.
   * For regular filter updates, use this method as a fallback function for the work item filter store methods instead.
   */
  updateFilterExpression: IPagePreferencesFilter["updateFilterExpression"] = (_workspaceSlug, id, filters) =>
    this.replace(id, "filters", () => toProjectFilters(filters));

  updateFilters: IPagePreferencesFilter["updateFilters"] = async (workspaceSlug, id, type, filters) => {
    switch (type) {
      case EIssueFilterType.DISPLAY_FILTERS:
        return this.merge(id, "display_filters", (held) => {
          const update = filters as IIssueDisplayFilterOptions;
          const result: IssueListDisplayFilters = {
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
        return this.merge(id, "display_properties", (held) => ({ ...held, ...filters }));
      case EIssueFilterType.KANBAN_FILTERS: {
        const value = { ...this.filters[id]?.kanbanFilters, ...filters } as TIssueKanbanFilters;
        const userId = this.rootIssueStore.currentUserId;
        if (userId)
          this.handleIssuesLocalFilters.set(this.storeType, type, workspaceSlug, id, userId, {
            kanban_filters: value,
          });
        runInAction(() => set(this.filters, [id, "kanbanFilters"], value));
      }
    }
  };
}

function keep<K extends PagePreferenceField>(held: PagePreferences, value: PagePreferences, field: K) {
  value[field] = held[field];
}
