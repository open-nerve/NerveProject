/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueCreate, IssueDetail, IssueUpdate } from "@nerve/api-client";
import { makeObservable, observable, runInAction } from "mobx";
import type { TLoader } from "@nerve/types";
import { pagePartKey, type Group } from "@/lib/paged-groups";
import type { IssueListOptions } from "@/lib/issue-filters";
import { issueMembership } from "@/lib/issue-membership";
import { compareIssueTime } from "@/lib/issue-order";
import type { IssueDrop, IIssueRootStore } from "../root.store";
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
    return this.currentList?.value?.groups;
  }
  get total(): number | undefined {
    return this.currentList?.value?.total;
  }
  getIssueIds = (group = "", subgroup?: string): string[] | undefined =>
    this.currentList
      ?.group(group, subgroup)
      ?.rows.filter((row) => !!this.rootIssueStore.issues.getIssueById(row.id))
      .map((row) => row.id);
  getGroupIssueCount = (group?: string, subgroup?: string, cumulative = false): number | undefined =>
    cumulative && subgroup !== undefined
      ? this.currentList?.value?.subgroupTotals?.get(subgroup)
      : this.currentList?.group(group, subgroup)?.total;
  getPaginationData = (group?: string, subgroup?: string): Group | undefined =>
    this.currentList?.group(group, subgroup);
  getIssueLoader = (group?: string, subgroup?: string): TLoader =>
    this.currentList?.verifying
      ? "init-loader"
      : this.currentList?.loaders.get(group === undefined ? "whole" : pagePartKey({ group, subgroup }));
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
  dropIssue = async (
    slug: string,
    project: string,
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop: IssueDrop
  ): Promise<void> => {
    const binding = this.currentList;
    if (!binding) return;
    const location = { ...drop, binding };
    await this.rootIssueStore.updateIssue(
      id,
      (held) => {
        if (!binding.value) throw new Error("Drop source requires a current list read");
        const body = makeBody(held);
        // Calendar uses its date query; a day move does not reorder the issue.
        if (binding.scope.query.group_by === "target_date") return body;
        const group = binding.group(location.group, location.subgroup);
        const legal = (neighborID: string | undefined) => {
          if (!neighborID || neighborID === id) return undefined;
          const neighbor = this.rootIssueStore.issues.getIssueById(neighborID);
          if (!neighbor) {
            // A hidden/unverified entity is not evidence of an empty destination; a tombstone is.
            if (this.rootIssueStore.issues.held(neighborID)) throw new Error("Drop neighbor requires verification");
            return undefined;
          }
          const part =
            location.subgroup === undefined
              ? { kind: "group" as const, group: location.group }
              : { kind: "cell" as const, group: location.group, subgroup: location.subgroup };
          const membership = issueMembership(neighbor, binding.context, part);
          if (membership !== undefined) return membership ? neighbor : undefined;
          // Unknown relations cannot be synthesized from Issue fields. Fresh rows under this immutable
          // query are server evidence: uncertain confirmations retain stale rows instead of inventing them.
          const row = group?.rows.find((item) => item.id === neighborID);
          if (
            !group?.stale &&
            !binding.error &&
            !binding.verifying &&
            row &&
            compareIssueTime(row.updatedAt, neighbor.updated_at) === 0
          )
            return neighbor;
          throw new Error("Drop neighbor requires a current list read");
        };
        const rows = group?.rows ?? [];
        const neighbors = rows
          .map((row) => legal(row.id))
          .filter((issue): issue is Issue => !!issue)
          .toSorted((a, b) => a.sort_order - b.sort_order || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
        let before = legal(location.beforeId);
        let after = legal(location.afterId);
        if (!before && !after) {
          if (location.beforeId && !location.afterId) before = neighbors[0];
          else after = neighbors.at(-1);
        } else if (!before && location.beforeId && after) {
          before = neighbors.find((item) => item.sort_order > after!.sort_order);
        } else if (!after && location.afterId && before) {
          after = neighbors.findLast((item) => item.sort_order < before!.sort_order);
        }
        body.sort_order =
          before && after
            ? (before.sort_order + after.sort_order) / 2
            : before
              ? before.sort_order - 65535
              : after
                ? after.sort_order + 65535
                : 65535;
        return body;
      },
      location,
      this.rootIssueStore.scope(slug, project)
    );
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
