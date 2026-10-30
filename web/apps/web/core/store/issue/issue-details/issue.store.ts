/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { Issue, IssueDetail, IssueUpdate } from "@nerve/api-client";
import { computedFn } from "mobx-utils";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { DetailResource, type DetailRead } from "./resource";
import type { IIssueDetail } from "./root.store";

export interface IIssueStoreActions {
  fetchIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<IssueDetail | undefined>;
  updateIssue: (workspaceSlug: string, projectId: string, issueId: string, data: IssueUpdate) => Promise<void>;
  removeIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  archiveIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  fetchIssueWithIdentifier: (workspaceSlug: string, identifier: string) => Promise<IssueDetail | undefined>;
}
export interface IIssueStore extends IIssueStoreActions {
  getIssueById: (issueId: string) => Issue | IssueDetail | undefined;
  getIssueDetailById: (issueId: string) => IssueDetail | undefined;
  getIssueIdByIdentifier: (issueIdentifier: string) => string | undefined;
  fetchIssueCore: (workspaceSlug: string, projectId: string, issueId: string) => Promise<IssueDetail | undefined>;
}

export class IssueStore implements IIssueStore {
  // Entries hold only IDs and read proofs. IssueRootStore.issues owns the single entity.
  private readonly reads = new Map<string, DetailResource<string>>();
  constructor(readonly rootIssueDetailStore: IIssueDetail) {}
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }
  getIssueById = computedFn((id: string) => this.root.issues.getIssueById(id));
  getIssueDetailById = computedFn((id: string) => this.root.issues.getIssueDetailById(id));
  getIssueIdByIdentifier = computedFn((identifier: string) => {
    const id = this.root.issues.getIssueIdByIdentifier(identifier);
    return id && this.root.issues.getIssueById(id) ? id : undefined;
  });
  private resource(workspaceID: string, projectIDs: string[], key: string): DetailResource<string> {
    const identity = JSON.stringify([workspaceID, projectIDs, key]);
    let resource = this.reads.get(identity);
    if (!resource) {
      resource = new DetailResource(this.root, { workspaceID, projectIDs });
      this.reads.set(identity, resource);
    }
    return resource;
  }
  private async accept(issue: IssueDetail, read: DetailRead): Promise<string> {
    if (!read.isCurrent([issue.project_id])) throw new SessionChangedError();
    this.root.issues.acceptRead(issue, read.token);
    if (!this.root.issues.getIssueDetailById(issue.id)) await this.root.verification.refresh(issue.id, false);
    if (!read.isCurrent([issue.project_id])) throw new SessionChangedError();
    return issue.id;
  }
  fetchIssueCore = async (workspaceSlug: string, projectId: string, issueId: string) => {
    this.root.assertProjectReadable(workspaceSlug, projectId);
    const workspace = this.root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    const project = this.root.rootStore.projectRoot.project.getProjectById(projectId);
    if (!workspace || project?.workspace_id !== workspace.id) throw new SessionChangedError();
    const resource = this.resource(workspace.id, [projectId], issueId);
    const id = await resource.fetch(async (read) => {
      const issue = await this.root.api.get(issueId);
      if (issue.id !== issueId || issue.project_id !== projectId || issue.workspace_id !== workspace.id)
        throw new SessionChangedError();
      return this.accept(issue, read);
    });
    if (!id) throw new SessionChangedError();
    return this.root.issues.getIssueDetailById(id);
  };
  private async loadIssueResources(workspaceSlug: string, issue: Issue, current: () => void): Promise<void> {
    const detail = this.rootIssueDetailStore;
    const { id, project_id: projectId } = issue;
    const run = (request: () => Promise<unknown>) => {
      current();
      return request();
    };
    await Promise.all([
      run(() => detail.fetchReactions(workspaceSlug, projectId, id)),
      run(() => detail.fetchLinks(workspaceSlug, projectId, id)),
      ...(issue.parent_id ? [run(() => this.fetchIssueCore(workspaceSlug, projectId, issue.parent_id!))] : []),
    ]);
    current();
    await Promise.all([
      run(() => detail.activity.fetchActivities(workspaceSlug, projectId, id)),
      run(() => detail.comment.fetchComments(workspaceSlug, projectId, id)),
      run(() => detail.subIssues.fetchSubIssues(workspaceSlug, projectId, id)),
      run(() => detail.relation.fetchRelations(workspaceSlug, projectId, id)),
      run(() => this.root.rootStore.state.fetchProjectStates(projectId)),
    ]);
    current();
  }
  fetchIssue = async (workspaceSlug: string, projectId: string, issueId: string) => {
    const token = this.root.captureRead();
    const issue = await this.fetchIssueCore(workspaceSlug, projectId, issueId);
    if (!issue) return undefined;
    const current = () => {
      if (!this.root.isReadCurrent(token, [projectId], issue.workspace_id)) throw new SessionChangedError();
      this.root.assertProjectReadable(workspaceSlug, projectId);
    };
    try {
      await this.loadIssueResources(workspaceSlug, issue, current);
    } catch (error) {
      current();
      throw error;
    }
    return this.root.issues.getIssueDetailById(issueId);
  };
  fetchIssueWithIdentifier = async (workspaceSlug: string, identifier: string) => {
    this.root.assertActive();
    const workspace = this.root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    if (!workspace) throw new SessionChangedError();
    const resource = this.resource(workspace.id, [], identifier);
    const token = this.root.captureRead();
    const id = await resource.fetch(
      async (read) => {
        const issue = await this.root.api.getByIdentifier(workspaceSlug, identifier);
        if (issue.workspace_id !== workspace.id || !read.isCurrent([issue.project_id])) throw new SessionChangedError();
        const directory = this.root.rootStore.projectRoot.project;
        // The server parses the identifier; unknown projects then need current directory proof.
        if (!directory.getProjectById(issue.project_id)) await directory.fetchProject(issue.project_id);
        if (!read.isCurrent([issue.project_id])) throw new SessionChangedError();
        const project = directory.getProjectById(issue.project_id);
        if (project?.workspace_id !== workspace.id) throw new SessionChangedError();
        this.root.assertProjectReadable(workspaceSlug, project.id);
        return this.accept(issue, read);
      },
      (resourceID) => {
        const issue = this.root.issues.getIssueById(resourceID);
        return issue ? [issue.project_id] : [];
      }
    );
    if (!id) throw new SessionChangedError();
    const issue = this.root.issues.getIssueDetailById(id);
    if (!issue) return undefined;
    const current = () => {
      if (!this.root.isReadCurrent(token, [issue.project_id], workspace.id)) throw new SessionChangedError();
      this.root.assertProjectReadable(workspaceSlug, issue.project_id);
    };
    try {
      await this.loadIssueResources(workspaceSlug, issue, current);
    } catch (error) {
      current();
      throw error;
    }
    this.root.issues.addIssueIdentifier(identifier, id);
    return this.root.issues.getIssueDetailById(id);
  };
  updateIssue = async (workspaceSlug: string, projectId: string, issueId: string, data: IssueUpdate) => {
    const scope = this.root.scope(workspaceSlug, projectId);
    const token = this.root.captureRead();
    await this.root.projectIssues.updateIssue(workspaceSlug, projectId, issueId, data);
    if (!this.root.isReadCurrent(token, [projectId], scope.workspaceID)) throw new SessionChangedError();
  };
  removeIssue = (workspaceSlug: string, projectId: string, issueId: string) =>
    this.root.projectIssues.removeIssue(workspaceSlug, projectId, issueId);
  archiveIssue = (workspaceSlug: string, projectId: string, issueId: string) =>
    this.root.projectIssues.archiveIssue(workspaceSlug, projectId, issueId);
}
