/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueActivity } from "@nerve/api-client";
import type { TIssueActivityComment } from "@nerve/types";
import { EActivityFilterType, E_SORT_ORDER } from "@nerve/constants";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IIssueDetail } from "./root.store";
import { IssueResources } from "./resources";
import { compareActivityOrder } from "./activity-order";

type Activities = { data: IssueActivity[]; cursor?: string };
function merged(held: IssueActivity[], answer: IssueActivity[]): IssueActivity[] {
  const rows = new Map(held.map((row) => [row.id, row]));
  for (const row of answer) rows.set(row.id, row);
  return [...rows.values()].toSorted(compareActivityOrder);
}
export type IIssueActivityStore = IssueActivityStore;
export type IIssueActivityStoreActions = Pick<IssueActivityStore, "fetchActivities">;

export class IssueActivityStore {
  private readonly resources: IssueResources<Activities>;
  constructor(readonly detail: IIssueDetail) {
    this.resources = new IssueResources(detail.rootIssueStore);
  }
  getActivitiesByIssueId = (id: string) => this.resources.get(id)?.value?.data.map((row) => row.id);
  getActivityById = (id: string, activityId: string) =>
    this.resources.get(id)?.value?.data.find((row) => row.id === activityId);
  getError = (id: string) => this.resources.get(id)?.error;
  refreshAfterChange = async (ws: string, project: string, id: string): Promise<void> => {
    const resource = this.resources.get(id);
    if (!resource) return;
    await resource.refreshAfterConfirmation(
      () => this.fetchActivities(ws, project, id, true),
      this.detail.rootIssueStore.captureRead()
    );
  };
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchActivities(ws, project, id);
  };
  fetchActivities = async (ws: string, project: string, id: string, incremental = false): Promise<IssueActivity[]> => {
    const resource = this.resources.entry(ws, project, id);
    const before = resource.value;
    const value = await resource.fetch(async (read) => {
      let rows: IssueActivity[] = [];
      const seen = new Set<string>();
      const pageFrom = async (cursor: string | undefined): Promise<Activities> => {
        const page = await this.detail.rootIssueStore.activityApi.getIssueActivities(
          id,
          cursor === undefined ? {} : { cursor }
        );
        if (!read.isCurrent()) throw new SessionChangedError();
        if (page.data.some((row) => row.issue_id !== id || row.project_id !== project)) throw new SessionChangedError();
        rows = merged(rows, page.data);
        if (page.next_cursor === null) return { data: merged(incremental ? (before?.data ?? []) : [], rows), cursor };
        if (seen.has(page.next_cursor)) throw new Error("Repeated activity cursor");
        seen.add(page.next_cursor);
        return pageFrom(page.next_cursor);
      };
      return pageFrom(incremental ? before?.cursor : undefined);
    });
    if (!value) throw new SessionChangedError();
    return value.data;
  };
  getActivityAndCommentsByIssueId = (id: string, order: E_SORT_ORDER): TIssueActivityComment[] | undefined => {
    const activities = this.resources.get(id)?.value?.data;
    const comment = this.detail.comment;
    const ids = comment.getCommentsByIssueId(id);
    if (!activities || !ids) return undefined;
    const rows: TIssueActivityComment[] = activities.map((row) => ({
      id: row.id,
      kind: "activity",
      created_at: row.created_at,
      activity_type:
        row.field === "state"
          ? EActivityFilterType.STATE
          : row.field === "assignees"
            ? EActivityFilterType.ASSIGNEE
            : row.field === null
              ? EActivityFilterType.DEFAULT
              : EActivityFilterType.ACTIVITY,
    }));
    for (const commentId of ids) {
      const row = comment.getCommentById(id, commentId);
      if (row)
        rows.push({
          id: row.id,
          kind: "comment",
          created_at: row.created_at,
          activity_type: EActivityFilterType.COMMENT,
        });
    }
    const sorted = rows.toSorted(compareActivityOrder);
    return order === E_SORT_ORDER.DESC ? sorted.toReversed() : sorted;
  };
}
