/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueRelation, IssueRelationType } from "@nerve/api-client";
import { computedFn } from "mobx-utils";
import { REVERSE_RELATIONS } from "@nerve/constants";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueReadToken, IssueWriteScope } from "../root.store";
import type { IIssueDetail } from "./root.store";
import { IssueResources } from "./resources";

export type RelationRow = { relationId: string; type: IssueRelationType; targetId: string; createdAt: string };
export type IIssueRelationStore = IssueRelationStore;
export type IIssueRelationStoreActions = Pick<
  IssueRelationStore,
  "fetchRelations" | "createRelation" | "removeRelation"
>;
const rowsOf = (items: IssueRelation[]): RelationRow[] => [
  ...new Map(
    items.map((item) => [
      item.id,
      { relationId: item.id, type: item.relation_type, targetId: item.issue.id, createdAt: item.created_at },
    ])
  ).values(),
];
const merged = (held: RelationRow[] | undefined, incoming: RelationRow[]) => [
  ...new Map([...(held ?? []), ...incoming].map((row) => [row.relationId, row])).values(),
];

export class IssueRelationStore {
  private readonly resources: IssueResources<RelationRow[]>;
  constructor(readonly rootIssueDetailStore: IIssueDetail) {
    this.resources = new IssueResources(this.root);
  }
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }
  getRelationRows = (id: string) =>
    this.resources.get(id)?.value?.filter((row) => this.root.issues.held(row.targetId) !== null);
  getRelationsByIssueId = computedFn((id: string): Record<IssueRelationType, string[]> | undefined => {
    const rows = this.getRelationRows(id);
    if (!rows) return undefined;
    const groups: Record<IssueRelationType, string[]> = { blocking: [], blocked_by: [], duplicate: [], relates_to: [] };
    for (const row of rows) groups[row.type].push(row.targetId);
    return groups;
  });
  getRelationCountByIssueId = (id: string, options: Partial<Record<IssueRelationType, unknown>>) =>
    this.getRelationRows(id)?.filter((row) => options[row.type]).length ?? 0;
  getError = (id: string) => this.resources.get(id)?.error;
  getProjectId = (id: string) => this.resources.get(id)?.scope.projectIDs[0];
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchRelations(ws, project, id);
  };
  private current(scope: IssueWriteScope, token: IssueReadToken, projects: string[]) {
    if (!this.root.isReadCurrent(token, [scope.projectID, ...projects], scope.workspaceID))
      throw new SessionChangedError();
    this.root.assertProjectReadable(scope.workspaceSlug, scope.projectID);
  }
  private async prove(items: IssueRelation[], scope: IssueWriteScope, token: IssueReadToken, isCurrent: () => boolean) {
    const projects = [...new Set(items.map((item) => item.issue.project_id))];
    const check = () => {
      this.current(scope, token, projects);
      if (!isCurrent() || items.some((item) => item.issue.workspace_id !== scope.workspaceID))
        throw new SessionChangedError();
    };
    check();
    for (const project of projects) {
      if (!this.root.rootStore.projectRoot.project.getProjectById(project))
        // oxlint-disable-next-line no-await-in-loop
        await this.root.rootStore.projectRoot.project.fetchProject(project);
      check();
      this.root.assertProjectReadable(scope.workspaceSlug, project);
    }
    return projects;
  }
  private async directories(projects: string[], scope: IssueWriteScope) {
    const app = this.root.rootStore;
    const otherProjects = projects.filter((project) => project !== scope.projectID);
    if (!otherProjects.length) return;
    const answers = await Promise.all([
      app.state.fetchWorkspaceStates({ id: scope.workspaceID, slug: scope.workspaceSlug }),
      ...otherProjects.flatMap((project) => [
        app.memberRoot.project.fetchProjectMembers(project),
        app.label.fetchProjectLabels(project),
      ]),
    ]);
    if (answers.some((answer) => answer === undefined)) throw new Error("Directory read was superseded");
  }
  fetchRelations = async (ws: string, project: string, id: string): Promise<RelationRow[]> => {
    const resource = this.resources.entry(ws, project, id);
    const scope = this.root.scope(ws, project);
    const answer = await resource.fetch(
      async (read) => {
        const result = await this.root.detailApi.listRelations(id);
        const projects = await this.prove(result.data, scope, read.token, () => read.isCurrent());
        await this.directories(projects, scope);
        this.current(scope, read.token, projects);
        if (!read.isCurrent()) throw new SessionChangedError();
        this.root.acceptListIssues(
          result.data.map((item) => item.issue),
          read.token
        );
        return rowsOf(result.data);
      },
      (rows) =>
        rows.flatMap((row) => {
          const target = this.root.issues.held(row.targetId);
          return target ? [target.project_id] : [];
        })
    );
    if (!answer) throw new SessionChangedError();
    return answer;
  };
  private async refreshEndpoints(ids: string[], scope: IssueWriteScope, token: IssueReadToken, projects: string[]) {
    this.current(scope, token, projects);
    const results = await Promise.allSettled(
      [...new Set(ids)].map((id) => this.root.issues.refreshAuthoritative(id, { afterWrite: true }))
    );
    this.current(scope, token, projects);
    const failed = results.find((result) => result.status === "rejected");
    if (failed?.status === "rejected") throw failed.reason;
    await Promise.all(
      [...new Set(ids)].map(async (id) => {
        const issue = this.root.issues.getIssueById(id);
        if (issue) await this.rootIssueDetailStore.refreshAfterChange(scope.workspaceSlug, issue.project_id, id);
      })
    );
  }
  createRelation = (ws: string, project: string, id: string, type: IssueRelationType, ids: string[]): Promise<void> =>
    this.root.enqueueIssueChanges([id, ...ids], async () => {
      const resource = this.resources.entry(ws, project, id);
      const scope = this.root.scope(ws, project);
      const token = this.root.captureRead();
      const answer = await this.root.detailApi.createRelations(id, { relation_type: type, issue_ids: ids });
      this.current(
        scope,
        token,
        answer.data.map((row) => row.issue.project_id)
      );
      let first = true;
      let confirmed = false;
      const projects = [...new Set(answer.data.map((row) => row.issue.project_id))];
      await resource.refreshAfterConfirmation(async () => {
        if (first) {
          first = false;
          await this.prove(answer.data, scope, token, () => this.root.active);
          this.root.acceptListIssues(
            answer.data.map((row) => row.issue),
            token
          );
          resource.receive(rowsOf(answer.data), token, merged, projects);
          for (const item of answer.data)
            this.resources.get(item.issue.id)?.receive(
              [
                {
                  relationId: item.id,
                  type: REVERSE_RELATIONS[item.relation_type],
                  targetId: id,
                  createdAt: item.created_at,
                },
              ],
              token,
              merged,
              [project, item.issue.project_id]
            );
          confirmed = true;
        } else if (!confirmed) {
          await this.fetchRelations(ws, project, id);
          confirmed = true;
        }
        await this.refreshEndpoints([id, ...ids], scope, token, projects);
        await this.directories(projects, scope);
      }, token);
    });
  removeRelation = (ws: string, project: string, id: string, type: IssueRelationType, target: string): Promise<void> =>
    this.root.enqueueIssueChanges([id, target], async () => {
      const resource = this.resources.entry(ws, project, id);
      const row = this.getRelationRows(id)?.find((entry) => entry.type === type && entry.targetId === target);
      if (!row) throw new Error("Relation has not been read");
      const scope = this.root.scope(ws, project);
      const token = this.root.captureRead();
      const targetProject = this.root.issues.held(target)?.project_id;
      const projects = targetProject ? [project, targetProject] : [project];
      for (const involved of projects) this.root.assertProjectReadable(ws, involved);
      await this.root.detailApi.deleteRelation(row.relationId);
      this.current(scope, token, projects);
      const remove = (held: RelationRow[]) => held.filter((entry) => entry.relationId !== row.relationId);
      resource.confirm(remove, token);
      this.resources.get(target)?.confirm(remove, token);
      await resource.refreshAfterConfirmation(() => this.refreshEndpoints([id, target], scope, token, projects), token);
    });
}
