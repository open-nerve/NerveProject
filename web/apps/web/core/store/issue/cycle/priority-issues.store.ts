/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observable, runInAction } from "mobx";
import type { PagedGroups } from "@/lib/paged-groups";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { IssueListBinding } from "../helpers/issue-list-binding";
import type { IIssueRootStore } from "../root.store";

/** A cycle in progress, as its card names it: the cycle and the project it belongs to. */
// Deliberately not the shared PlanningPage: the priority list is cycle-only (P6b spec 2.2); entityID would blur it.
type CyclePage = Readonly<{ projectID: string; cycleID: string }>;

/**
 * What the tab shows of a cycle's priority work items once read: nerve's count of them, the rows the session may see
 * in nerve's order, whether a next page is left, and why reading one failed, if it did.
 */
type PriorityIssuesView = Readonly<{ total: number; issueIds: string[]; more: boolean; error: unknown }>;

/**
 * The query of a cycle's priority work items (P5b spec 2.7, R7): whole and explicit, none of a page's preferences in
 * it: the cycle's urgent and high ones, its sub-work items too, the newest first, thirty a page.
 */
const priorityQuery = (cycleID: string): ProjectIssueQuery => ({
  cycle_id: [cycleID],
  priority: ["urgent", "high"],
  order_by: "-created_at",
  sub_issue: "true",
  limit: 30,
});

/**
 * The priority work items of cycles, for the tab of the card of the cycle in progress (P5b spec 2.7, R7; re-66 M4): a
 * list store of their own beside the cycle page's, so that reading them never changes which list that page shows. The
 * root makes one as it makes the page's, and it ends with the root. Each cycle has one binding, made by its first read
 * and registered in the root as every list is (the root's lists only grow): the root's confirmations of writes reach
 * it, and its cycle and priorities decide whether a row stays. Reading what a cycle holds makes nothing.
 */
export class CyclePriorityIssues {
  /** Each cycle's binding, by its workspace's id (v0 design 7.7), its project and the cycle. */
  private readonly bindings = observable.map<string, IssueListBinding>({}, { deep: false });
  constructor(private readonly root: IIssueRootStore) {}

  private static key(workspaceID: string, page: CyclePage): string {
    return JSON.stringify([workspaceID, page.projectID, page.cycleID]);
  }
  /** The workspace slug names in the session's list of them. */
  private workspaceID(slug: string): string | undefined {
    return this.root.rootStore.workspaceRoot.getWorkspaceBySlug(slug)?.id;
  }
  /** The cycle's binding, in the workspace slug names now, once a read made it. */
  private list(slug: string, page: CyclePage): IssueListBinding | undefined {
    const workspaceID = this.workspaceID(slug);
    return workspaceID === undefined ? undefined : this.bindings.get(CyclePriorityIssues.key(workspaceID, page));
  }
  /**
   * The caller's permissions in the cycle's project as the root last saw them: a read under others is read again
   * (use-issue-list-mount.ts).
   */
  readIdentity(slug: string, page: CyclePage): string {
    return this.root.permissionReads.identity(this.workspaceID(slug), [page.projectID]);
  }
  /** Reads the first page of the cycle's priority work items, its binding made by the first read. */
  fetchIssues = (slug: string, page: CyclePage): Promise<PagedGroups | undefined> => {
    const scope = this.root.scope(slug, page.projectID);
    const key = CyclePriorityIssues.key(scope.workspaceID, page);
    let list = this.bindings.get(key);
    if (!list) {
      const made = new IssueListBinding(this.root, { ...scope, scope: "project", query: priorityQuery(page.cycleID) });
      runInAction(() => this.bindings.set(key, made));
      list = made;
    }
    return list.fetchFirst();
  };
  /** Reads the cycle's next page by its cursor, after the rows held, in nerve's order: never the cycle page's (R7). */
  fetchNextIssues = (slug: string, page: CyclePage): Promise<PagedGroups | undefined> =>
    this.list(slug, page)?.fetchNext() ?? Promise.resolve(undefined);
  /** Reads again what failed of the cycle's list. */
  retryIssues = (slug: string, page: CyclePage): Promise<void> =>
    this.list(slug, page)?.retryFailed() ?? Promise.resolve();
  /** What the tab shows of the cycle (PriorityIssuesView); nothing until its first read is in. */
  view(slug: string, page: CyclePage): PriorityIssuesView | undefined {
    const list = this.list(slug, page);
    const value = list?.value;
    const rows = list?.group();
    if (!list || !value || !rows) return undefined;
    return {
      total: value.total,
      issueIds: rows.rows.filter((row) => !!this.root.issues.getIssueById(row.id)).map((row) => row.id),
      more: !!rows.nextCursor,
      error: list.error,
    };
  }
}
