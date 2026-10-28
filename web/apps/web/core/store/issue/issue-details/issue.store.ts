/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { TLegacyIssueUpdateRequest } from "@nerve/types";
import { computedFn } from "mobx-utils";
// types

// services
import { IssueService } from "@/services/issue";
// types
import type { IIssueDetail } from "./root.store";

export interface IIssueStoreActions {
  // actions
  fetchIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<IssueDetail>;
  updateIssue: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: TLegacyIssueUpdateRequest
  ) => Promise<void>;
  removeIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  archiveIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  fetchIssueWithIdentifier: (
    workspaceSlug: string,
    project_identifier: string,
    sequence_id: string
  ) => Promise<IssueDetail>;
}

export interface IIssueStore extends IIssueStoreActions {
  // helper methods
  getIssueById: (issueId: string) => Issue | IssueDetail | undefined;
  getIssueDetailById: (issueId: string) => IssueDetail | undefined;
  getIssueIdByIdentifier: (issueIdentifier: string) => string | undefined;
}

export class IssueStore implements IIssueStore {
  // root store
  rootIssueDetailStore: IIssueDetail;
  // services
  issueService;

  constructor(rootStore: IIssueDetail) {
    // root store
    this.rootIssueDetailStore = rootStore;
    // services
    this.issueService = new IssueService();
  }

  // helper methods
  getIssueById = computedFn((issueId: string): Issue | IssueDetail | undefined => {
    if (!issueId) return undefined;
    return this.rootIssueDetailStore.rootIssueStore.issues.getIssueById(issueId) ?? undefined;
  });

  getIssueDetailById = computedFn((issueId: string) =>
    this.rootIssueDetailStore.rootIssueStore.issues.getIssueDetailById(issueId)
  );

  getIssueIdByIdentifier = computedFn((issueIdentifier: string) => {
    if (!issueIdentifier) return undefined;
    return this.rootIssueDetailStore.rootIssueStore.issues.getIssueIdByIdentifier(issueIdentifier) ?? undefined;
  });

  private async loadIssueResources(workspaceSlug: string, projectId: string, issueId: string, issue: Issue) {
    // Auxiliary resources keep their own DTOs; they are not fields of the core detail response.
    await Promise.all([
      this.rootIssueDetailStore.fetchReactions(workspaceSlug, projectId, issueId),
      this.rootIssueDetailStore.fetchLinks(workspaceSlug, projectId, issueId),
      ...(issue.parent_id && issue.project_id
        ? [
            this.issueService
              .retrieve(workspaceSlug, issue.project_id, issue.parent_id)
              .then((parent) => this.rootIssueDetailStore.rootIssueStore.issues.addIssue([parent])),
          ]
        : []),
    ]);
  }

  // actions
  fetchIssue = async (workspaceSlug: string, projectId: string, issueId: string) => {
    const issue = await this.issueService.retrieve(workspaceSlug, projectId, issueId);

    if (!issue) throw new Error("Work item not found");

    this.addIssueToStore(issue);

    await this.loadIssueResources(workspaceSlug, projectId, issueId, issue);

    // fetch issue activity
    this.rootIssueDetailStore.activity.fetchActivities(workspaceSlug, projectId, issueId);

    // fetch issue comments
    this.rootIssueDetailStore.comment.fetchComments(workspaceSlug, projectId, issueId);

    // fetch sub issues
    this.rootIssueDetailStore.subIssues.fetchSubIssues(workspaceSlug, projectId, issueId);

    // fetch issue relations
    this.rootIssueDetailStore.relation.fetchRelations(workspaceSlug, projectId, issueId);

    // fetching states
    // TODO: check if this function is required
    this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);

    return issue;
  };

  addIssueToStore = (issue: Issue) => {
    this.rootIssueDetailStore.rootIssueStore.issues.addIssue([issue]);

    return issue;
  };

  updateIssue = async (workspaceSlug: string, projectId: string, issueId: string, data: TLegacyIssueUpdateRequest) => {
    const currentStore = this.rootIssueDetailStore.rootIssueStore.projectIssues;

    await Promise.all([
      currentStore.updateIssue(workspaceSlug, projectId, issueId, data),
      this.rootIssueDetailStore.activity.fetchActivities(workspaceSlug, projectId, issueId),
    ]);
  };

  removeIssue = async (workspaceSlug: string, projectId: string, issueId: string) => {
    const currentStore = this.rootIssueDetailStore.rootIssueStore.projectIssues;
    currentStore.removeIssue(workspaceSlug, projectId, issueId);
  };

  archiveIssue = async (workspaceSlug: string, projectId: string, issueId: string) => {
    const currentStore = this.rootIssueDetailStore.rootIssueStore.projectIssues;
    currentStore.archiveIssue(workspaceSlug, projectId, issueId);
  };

  fetchIssueWithIdentifier = async (workspaceSlug: string, project_identifier: string, sequence_id: string) => {
    const issue = await this.issueService.retrieveWithIdentifier(workspaceSlug, project_identifier, sequence_id);
    const issueIdentifier = `${project_identifier}-${sequence_id}`;
    const issueId = issue?.id;
    const projectId = issue?.project_id;
    const rootWorkItemDetailStore = this.rootIssueDetailStore.rootIssueStore.issueDetail;

    if (!issue || !projectId || !issueId) throw new Error("Issue not found");

    this.addIssueToStore(issue);

    await this.loadIssueResources(workspaceSlug, projectId, issueId, issue);

    // add identifiers to map
    rootWorkItemDetailStore.rootIssueStore.issues.addIssueIdentifier(issueIdentifier, issueId);

    // fetch issue activity
    rootWorkItemDetailStore.activity.fetchActivities(workspaceSlug, projectId, issueId);

    // fetch issue comments
    rootWorkItemDetailStore.comment.fetchComments(workspaceSlug, projectId, issueId);

    // fetch sub issues
    rootWorkItemDetailStore.subIssues.fetchSubIssues(workspaceSlug, projectId, issueId);

    // fetch issue relations
    rootWorkItemDetailStore.relation.fetchRelations(workspaceSlug, projectId, issueId);

    // fetching states
    // TODO: check if this function is required
    rootWorkItemDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);

    return issue;
  };
}
