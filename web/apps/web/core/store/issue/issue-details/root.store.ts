/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueUpdate, IssueLinkCreate, IssueLinkPatch, IssueCommentCreate } from "@nerve/api-client";
import { action, makeObservable, observable } from "mobx";
// types
import type { TIssueRelationTypes, TWorkItemWidgets } from "@nerve/types";
// local imports
import { DescriptionVersionsStore } from "./description-versions.store";
import { IssueActivityStore } from "./activity.store";
import type { IIssueActivityStore, IIssueActivityStoreActions } from "./activity.store";
import type { IIssueRootStore } from "../root.store";
import { IssueCommentStore } from "./comment.store";
import type { IIssueCommentStore, IIssueCommentStoreActions } from "./comment.store";
import { IssueCommentReactionStore } from "./comment_reaction.store";
import type { IIssueCommentReactionStore, IIssueCommentReactionStoreActions } from "./comment_reaction.store";
import { IssueStore } from "./issue.store";
import type { IIssueStore, IIssueStoreActions } from "./issue.store";
import { IssueLinkStore } from "./link.store";
import type { IIssueLinkStore, IIssueLinkStoreActions } from "./link.store";
import { IssueReactionStore } from "./reaction.store";
import type { IIssueReactionStore, IIssueReactionStoreActions } from "./reaction.store";
import { IssueRelationStore } from "./relation.store";
import type { IIssueRelationStore, IIssueRelationStoreActions } from "./relation.store";
import { IssueSubIssuesStore } from "./sub_issues.store";
import type { IIssueSubIssuesStore, IIssueSubIssuesStoreActions } from "./sub_issues.store";

export type TPeekIssue = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  nestingLevel?: number;
  isArchived?: boolean;
};

export interface IIssueDetail
  extends
    IIssueStoreActions,
    IIssueReactionStoreActions,
    IIssueLinkStoreActions,
    IIssueSubIssuesStoreActions,
    IIssueRelationStoreActions,
    IIssueActivityStoreActions,
    IIssueCommentStoreActions,
    IIssueCommentReactionStoreActions {
  refreshAfterChange: (ws: string, project: string, id: string) => Promise<void>;
  // observables
  peekIssue: TPeekIssue | undefined;
  openWidgets: TWorkItemWidgets[];
  // computed
  // helper actions
  getIsIssuePeeked: (issueId: string) => boolean;
  // actions
  setPeekIssue: (peekIssue: TPeekIssue | undefined) => void;
  setLastWidgetAction: (action: TWorkItemWidgets) => void;
  toggleOpenWidget: (state: TWorkItemWidgets) => void;
  // store
  rootIssueStore: IIssueRootStore;
  issue: IIssueStore;
  reaction: IIssueReactionStore;
  activity: IIssueActivityStore;
  versions: DescriptionVersionsStore;
  comment: IIssueCommentStore;
  commentReaction: IIssueCommentReactionStore;
  subIssues: IIssueSubIssuesStore;
  link: IIssueLinkStore;
  relation: IIssueRelationStore;
}

export class IssueDetail implements IIssueDetail {
  // observables
  peekIssue: TPeekIssue | undefined = undefined;
  openWidgets: TWorkItemWidgets[] = ["sub-work-items", "links"];
  // store
  rootIssueStore: IIssueRootStore;
  issue: IIssueStore;
  reaction: IIssueReactionStore;
  subIssues: IIssueSubIssuesStore;
  link: IIssueLinkStore;
  relation: IIssueRelationStore;
  activity: IIssueActivityStore;
  versions: DescriptionVersionsStore;
  comment: IIssueCommentStore;
  commentReaction: IIssueCommentReactionStore;

  constructor(rootStore: IIssueRootStore) {
    makeObservable(this, {
      // observables
      peekIssue: observable.ref,
      openWidgets: observable.ref,
      // computed
      // action
      setPeekIssue: action,
      setLastWidgetAction: action,
      toggleOpenWidget: action,
    });

    // store
    this.rootIssueStore = rootStore;
    this.issue = new IssueStore(this);
    this.reaction = new IssueReactionStore(this);
    this.versions = new DescriptionVersionsStore(this);
    this.activity = new IssueActivityStore(this);
    this.comment = new IssueCommentStore(this);
    this.commentReaction = new IssueCommentReactionStore(this);
    this.subIssues = new IssueSubIssuesStore(this);
    this.link = new IssueLinkStore(this);
    this.relation = new IssueRelationStore(this);
  }

  // helper actions
  getIsIssuePeeked = (issueId: string) => this.peekIssue?.issueId === issueId;

  // actions
  setPeekIssue = (peekIssue: TPeekIssue | undefined) => (this.peekIssue = peekIssue);
  setLastWidgetAction = (widget: TWorkItemWidgets) => {
    this.openWidgets = [widget];
  };
  toggleOpenWidget = (state: TWorkItemWidgets) => {
    if (this.openWidgets && this.openWidgets.includes(state))
      this.openWidgets = this.openWidgets.filter((s) => s !== state);
    else this.openWidgets = [state, ...this.openWidgets];
  };

  // issue
  fetchIssue = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.issue.fetchIssue(workspaceSlug, projectId, issueId);
  fetchIssueWithIdentifier = async (workspaceSlug: string, identifier: string) =>
    this.issue.fetchIssueWithIdentifier(workspaceSlug, identifier);
  updateIssue = async (workspaceSlug: string, projectId: string, issueId: string, data: IssueUpdate) =>
    this.issue.updateIssue(workspaceSlug, projectId, issueId, data);
  removeIssue = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.issue.removeIssue(workspaceSlug, projectId, issueId);
  archiveIssue = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.issue.archiveIssue(workspaceSlug, projectId, issueId);
  refreshAfterChange = async (ws: string, project: string, id: string): Promise<void> => {
    await Promise.all([
      this.activity.refreshAfterChange(ws, project, id),
      this.comment.refreshAfterChange(ws, project, id),
    ]);
  };
  // reactions
  fetchReactions = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.reaction.fetchReactions(workspaceSlug, projectId, issueId);
  createReaction = async (workspaceSlug: string, projectId: string, issueId: string, reaction: string) =>
    this.reaction.createReaction(workspaceSlug, projectId, issueId, reaction);
  removeReaction = async (workspaceSlug: string, projectId: string, issueId: string, reaction: string) =>
    this.reaction.removeReaction(workspaceSlug, projectId, issueId, reaction);

  // link
  fetchLinks = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.link.fetchLinks(workspaceSlug, projectId, issueId);
  createLink = async (workspaceSlug: string, projectId: string, issueId: string, data: IssueLinkCreate) =>
    this.link.createLink(workspaceSlug, projectId, issueId, data);
  updateLink = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    linkId: string,
    data: IssueLinkPatch
  ) => this.link.updateLink(workspaceSlug, projectId, issueId, linkId, data);
  removeLink = async (workspaceSlug: string, projectId: string, issueId: string, linkId: string) =>
    this.link.removeLink(workspaceSlug, projectId, issueId, linkId);

  // sub issues
  fetchSubIssues = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.subIssues.fetchSubIssues(workspaceSlug, projectId, issueId);
  createSubIssues = async (workspaceSlug: string, projectId: string, parentIssueId: string, data: string[]) =>
    this.subIssues.createSubIssues(workspaceSlug, projectId, parentIssueId, data);
  updateSubIssue = async (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issueId: string,
    issueData: IssueUpdate
  ) => this.subIssues.updateSubIssue(workspaceSlug, projectId, parentIssueId, issueId, issueData);
  removeSubIssue = async (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) =>
    this.subIssues.removeSubIssue(workspaceSlug, projectId, parentIssueId, issueId);
  deleteSubIssue = async (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) =>
    this.subIssues.deleteSubIssue(workspaceSlug, projectId, parentIssueId, issueId);

  // relations
  fetchRelations = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.relation.fetchRelations(workspaceSlug, projectId, issueId);
  createRelation = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    issues: string[]
  ) => this.relation.createRelation(workspaceSlug, projectId, issueId, relationType, issues);
  removeRelation = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    relatedIssue: string
  ) => this.relation.removeRelation(workspaceSlug, projectId, issueId, relationType, relatedIssue);

  // activity
  fetchActivities = async (workspaceSlug: string, projectId: string, issueId: string, incremental = false) =>
    this.activity.fetchActivities(workspaceSlug, projectId, issueId, incremental);

  // comment
  fetchComments = async (workspaceSlug: string, projectId: string, issueId: string, incremental = false) =>
    this.comment.fetchComments(workspaceSlug, projectId, issueId, incremental);
  createComment = async (workspaceSlug: string, projectId: string, issueId: string, data: IssueCommentCreate) =>
    this.comment.createComment(workspaceSlug, projectId, issueId, data);
  updateComment = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string,
    data: IssueCommentCreate
  ) => this.comment.updateComment(workspaceSlug, projectId, issueId, commentId, data);
  removeComment = async (workspaceSlug: string, projectId: string, issueId: string, commentId: string) =>
    this.comment.removeComment(workspaceSlug, projectId, issueId, commentId);

  // comment reactions are owned by their comments, never a second fetched collection.
  createCommentReaction = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string,
    reaction: string
  ) => this.commentReaction.createCommentReaction(workspaceSlug, projectId, issueId, commentId, reaction);
  removeCommentReaction = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string,
    reaction: string
  ) => this.commentReaction.removeCommentReaction(workspaceSlug, projectId, issueId, commentId, reaction);
}
