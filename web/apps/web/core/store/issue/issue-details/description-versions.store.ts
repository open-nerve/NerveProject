/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { DescriptionVersion, DescriptionVersionDetail, DescriptionVersionPage } from "@nerve/api-client";
import { observable, runInAction } from "mobx";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { compareCodePoints, compareIssueTime } from "@/lib/issue-order";
import type { IIssueDetail } from "./root.store";
import { IssueResources } from "./resources";
import { DetailResource } from "./resource";
function ordered(rows: DescriptionVersion[]): DescriptionVersion[] {
  return [...new Map(rows.map((row) => [row.id, row])).values()].toSorted(
    (a, b) => compareIssueTime(b.last_saved_at, a.last_saved_at) || compareCodePoints(b.id, a.id)
  );
}
export class DescriptionVersionsStore {
  private readonly lists: IssueResources<DescriptionVersionPage>;
  private readonly details = observable.map<string, DetailResource<DescriptionVersionDetail>>({}, { deep: false });
  constructor(readonly detail: IIssueDetail) {
    this.lists = new IssueResources(detail.rootIssueStore, true);
  }
  canRead = (ws: string, project: string): boolean => {
    const root = this.detail.rootIssueStore;
    const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(ws);
    const role = root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(ws, project);
    return root.active && !!workspace && workspace.role !== 5 && (role === 15 || role === 20);
  };
  getVersions = (id: string) => this.lists.get(id)?.value?.data;
  hasMore = (id: string) => this.lists.get(id)?.value?.next_cursor != null;
  getError = (id: string) => this.lists.get(id)?.error;
  private key = (id: string, version: string) => JSON.stringify([id, version]);
  getVersion = (id: string, version: string) => this.details.get(this.key(id, version))?.value;
  fetchVersions = async (ws: string, project: string, id: string, next = false): Promise<DescriptionVersion[]> => {
    const resource = this.lists.entry(ws, project, id);
    const before = resource.value;
    if (next && before?.next_cursor === null) return before.data;
    const cursor = next ? (before?.next_cursor ?? undefined) : undefined;
    const value = await resource.fetch(async (read) => {
      const page = await this.detail.rootIssueStore.versionApi.listDescriptionVersions(id, cursor ? { cursor } : {});
      if (!read.isCurrent() || page.data.some((row) => row.issue_id !== id)) throw new SessionChangedError();
      if (cursor && page.next_cursor === cursor) throw new Error("Repeated description version cursor");
      return { data: ordered([...(next ? (before?.data ?? []) : []), ...page.data]), next_cursor: page.next_cursor };
    });
    if (!value) throw new SessionChangedError();
    return value.data;
  };
  fetchVersion = async (
    ws: string,
    project: string,
    id: string,
    version: string
  ): Promise<DescriptionVersionDetail> => {
    const list = this.lists.entry(ws, project, id);
    const key = this.key(id, version);
    let resource = this.details.get(key);
    if (!resource) {
      resource = new DetailResource(this.detail.rootIssueStore, list.scope);
      runInAction(() => this.details.set(key, resource!));
    }
    const value = await resource.fetch(async () => {
      const answer = await this.detail.rootIssueStore.versionApi.retrieveDescriptionVersion(version);
      if (answer.issue_id !== id || answer.id !== version) throw new SessionChangedError();
      return answer;
    });
    if (!value) throw new SessionChangedError();
    return value;
  };
  restore = async (ws: string, project: string, id: string, html: string): Promise<string> => {
    if (!this.canRead(ws, project)) throw new SessionChangedError();
    this.lists.entry(ws, project, id);
    const root = this.detail.rootIssueStore;
    const scope = root.scope(ws, project);
    const token = root.captureRead(true);
    const assertCurrent = () => {
      if (!root.isReadCurrent(token, [project], scope.workspaceID) || !this.canRead(ws, project))
        throw new SessionChangedError();
    };
    const answer = await root.updateIssue(
      id,
      () => {
        assertCurrent();
        return { description_html: html };
      },
      undefined,
      scope
    );
    assertCurrent();
    return answer.description_html;
  };
}
