/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueCreate, IssueDetail, IssueUpdate } from "@nerve/api-client";
import { makeObservable, observable, runInAction } from "mobx";
import type { TLoader } from "@nerve/types";
import { pagePartKey, type Group } from "@/lib/paged-groups";
import type { IssueListOptions } from "@/lib/issue-filters";
import type { IIssueRootStore } from "../root.store";
import type { IBaseIssueFilterStore } from "./issue-filter-helper.store";
import { IssueListBinding, type IssueListScope } from "./issue-list-binding";

/** The layout's selection; each query keeps its own registered reconciliation value. */
export class BaseIssuesStore<TFilter extends IBaseIssueFilterStore = IBaseIssueFilterStore> {
  private readonly bindings = new Map<string, IssueListBinding>();
  currentList: IssueListBinding | undefined;
  protected options: IssueListOptions | undefined;
  constructor(
    readonly rootIssueStore: IIssueRootStore,
    readonly issueFilterStore: TFilter
  ) {
    makeObservable(this, { currentList: observable.ref });
  }
  protected select(scope: IssueListScope, options: IssueListOptions): IssueListBinding {
    const key = JSON.stringify(scope);
    let list = this.bindings.get(key);
    if (!list) {
      list = new IssueListBinding(this.rootIssueStore, structuredClone(scope));
      this.bindings.set(key, list);
    }
    this.options = options;
    runInAction(() => {
      this.currentList = list;
    });
    return list;
  }
  get groups(): Map<string, Group> | undefined {
    return this.currentList?.reconciled.value?.groups;
  }
  get total(): number | undefined {
    return this.currentList?.reconciled.value?.total;
  }
  getIssueIds = (group = "", subgroup?: string): string[] | undefined =>
    this.currentList?.group(group, subgroup)?.rows.map((row) => row.id);
  getGroupIssueCount = (group?: string, subgroup?: string, cumulative = false): number | undefined =>
    cumulative && subgroup !== undefined
      ? this.currentList?.reconciled.value?.subgroupTotals?.get(subgroup)
      : this.currentList?.group(group, subgroup)?.total;
  getPaginationData = (group?: string, subgroup?: string): Group | undefined =>
    this.currentList?.group(group, subgroup);
  getIssueLoader = (group?: string, subgroup?: string): TLoader =>
    this.currentList?.loaders.get(group === undefined ? "whole" : pagePartKey({ group, subgroup }));
  clear = (_clearOptions = true): void => {
    runInAction(() => {
      this.currentList = undefined;
    });
  };
  createIssue = (slug: string, project: string, body: IssueCreate): Promise<IssueDetail> =>
    this.rootIssueStore.createIssue(this.rootIssueStore.scope(slug, project), body);
  updateIssue = async (slug: string, project: string, id: string, body: IssueUpdate): Promise<void> => {
    await this.rootIssueStore.updateIssue(id, () => body, undefined, this.rootIssueStore.scope(slug, project));
  };
  removeIssue = (slug: string, project: string, id: string): Promise<void> =>
    this.rootIssueStore.deleteIssue(id, this.rootIssueStore.scope(slug, project));
  archiveIssue = async (slug: string, project: string, id: string): Promise<void> => {
    await this.rootIssueStore.archiveIssue(id, this.rootIssueStore.scope(slug, project));
  };
  restoreIssue = async (slug: string, project: string, id: string): Promise<void> => {
    await this.rootIssueStore.unarchiveIssue(id, this.rootIssueStore.scope(slug, project));
  };
}
