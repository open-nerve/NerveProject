/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueUpdate } from "@nerve/api-client";
import { action, computed, makeObservable, observable } from "mobx";
// types
import type {
  TIssueComment,
  TIssueCommentReaction,
  TIssueLink,
  TIssueReaction,
  TIssueRelationTypes,
  TWorkItemWidgets,
} from "@nerve/types";
// local imports
import { IssueActivityStore } from "./activity.store";
import type { IIssueActivityStore, IIssueActivityStoreActions, TActivityLoader } from "./activity.store";
import type { IIssueRootStore } from "../root.store";
import { IssueCommentStore } from "./comment.store";
import type { IIssueCommentStore, IIssueCommentStoreActions, TCommentLoader } from "./comment.store";
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

export type TIssueRelationModal = {
  issueId: string | null;
  relationType: TIssueRelationTypes | null;
};

type TIssueCrudState = { toggle: boolean; parentIssueId: string | undefined };

export type TIssueCrudOperationState = {
  create: TIssueCrudState;
  existing: TIssueCrudState;
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
  // observables
  peekIssue: TPeekIssue | undefined;
  relationKey: TIssueRelationTypes | null;
  issueLinkData: TIssueLink | null;
  issueCrudOperationState: TIssueCrudOperationState;
  openWidgets: TWorkItemWidgets[];
  isCreateIssueModalOpen: boolean;
  isIssueLinkModalOpen: boolean;
  isParentIssueModalOpen: string | null;
  isDeleteIssueModalOpen: string | null;
  isRelationModalOpen: TIssueRelationModal | null;
  isSubIssuesModalOpen: string | null;
  // computed
  isAnyModalOpen: boolean;
  // helper actions
  getIsIssuePeeked: (issueId: string) => boolean;
  // actions
  setPeekIssue: (peekIssue: TPeekIssue | undefined) => void;
  setIssueLinkData: (issueLinkData: TIssueLink | null) => void;
  toggleCreateIssueModal: (value: boolean) => void;
  toggleIssueLinkModal: (value: boolean) => void;
  toggleParentIssueModal: (issueId: string | null) => void;
  toggleDeleteIssueModal: (issueId: string | null) => void;
  toggleRelationModal: (issueId: string | null, relationType: TIssueRelationTypes | null) => void;
  toggleSubIssuesModal: (value: string | null) => void;
  setLastWidgetAction: (action: TWorkItemWidgets) => void;
  toggleOpenWidget: (state: TWorkItemWidgets) => void;
  setRelationKey: (relationKey: TIssueRelationTypes | null) => void;
  setIssueCrudOperationState: (state: TIssueCrudOperationState) => void;
  // store
  rootIssueStore: IIssueRootStore;
  issue: IIssueStore;
  reaction: IIssueReactionStore;
  activity: IIssueActivityStore;
  comment: IIssueCommentStore;
  commentReaction: IIssueCommentReactionStore;
  subIssues: IIssueSubIssuesStore;
  link: IIssueLinkStore;
  relation: IIssueRelationStore;
}

export class IssueDetail implements IIssueDetail {
  // observables
  peekIssue: TPeekIssue | undefined = undefined;
  relationKey: TIssueRelationTypes | null = null;
  issueLinkData: TIssueLink | null = null;
  issueCrudOperationState: TIssueCrudOperationState = {
    create: {
      toggle: false,
      parentIssueId: undefined,
    },
    existing: {
      toggle: false,
      parentIssueId: undefined,
    },
  };
  openWidgets: TWorkItemWidgets[] = ["sub-work-items", "links"];
  isCreateIssueModalOpen: boolean = false;
  isIssueLinkModalOpen: boolean = false;
  isParentIssueModalOpen: string | null = null;
  isDeleteIssueModalOpen: string | null = null;
  isRelationModalOpen: TIssueRelationModal | null = null;
  isSubIssuesModalOpen: string | null = null;
  // store
  rootIssueStore: IIssueRootStore;
  issue: IIssueStore;
  reaction: IIssueReactionStore;
  subIssues: IIssueSubIssuesStore;
  link: IIssueLinkStore;
  relation: IIssueRelationStore;
  activity: IIssueActivityStore;
  comment: IIssueCommentStore;
  commentReaction: IIssueCommentReactionStore;

  constructor(rootStore: IIssueRootStore) {
    makeObservable(this, {
      // observables
      peekIssue: observable,
      relationKey: observable,
      issueLinkData: observable,
      issueCrudOperationState: observable,
      isCreateIssueModalOpen: observable,
      isIssueLinkModalOpen: observable.ref,
      isParentIssueModalOpen: observable.ref,
      isDeleteIssueModalOpen: observable.ref,
      isRelationModalOpen: observable.ref,
      isSubIssuesModalOpen: observable.ref,
      openWidgets: observable.ref,
      // computed
      isAnyModalOpen: computed,
      // action
      setPeekIssue: action,
      setIssueLinkData: action,
      toggleCreateIssueModal: action,
      toggleIssueLinkModal: action,
      toggleParentIssueModal: action,
      toggleDeleteIssueModal: action,
      toggleRelationModal: action,
      toggleSubIssuesModal: action,
      setLastWidgetAction: action,
      toggleOpenWidget: action,
      setRelationKey: action,
      setIssueCrudOperationState: action,
    });

    // store
    this.rootIssueStore = rootStore;
    this.issue = new IssueStore(this);
    this.reaction = new IssueReactionStore(this);
    this.activity = new IssueActivityStore(rootStore.rootStore);
    this.comment = new IssueCommentStore(this);
    this.commentReaction = new IssueCommentReactionStore(this);
    this.subIssues = new IssueSubIssuesStore(this);
    this.link = new IssueLinkStore(this);
    this.relation = new IssueRelationStore(this);
  }

  // computed
  get isAnyModalOpen() {
    return (
      this.isCreateIssueModalOpen ||
      this.isIssueLinkModalOpen ||
      !!this.isParentIssueModalOpen ||
      !!this.isDeleteIssueModalOpen ||
      !!this.isRelationModalOpen?.issueId ||
      !!this.isSubIssuesModalOpen
    );
  }

  // helper actions
  getIsIssuePeeked = (issueId: string) => this.peekIssue?.issueId === issueId;

  // actions
  setRelationKey = (relationKey: TIssueRelationTypes | null) => (this.relationKey = relationKey);
  setIssueCrudOperationState = (state: TIssueCrudOperationState) => (this.issueCrudOperationState = state);
  setPeekIssue = (peekIssue: TPeekIssue | undefined) => (this.peekIssue = peekIssue);
  toggleCreateIssueModal = (value: boolean) => (this.isCreateIssueModalOpen = value);
  toggleIssueLinkModal = (value: boolean) => (this.isIssueLinkModalOpen = value);
  toggleParentIssueModal = (issueId: string | null) => (this.isParentIssueModalOpen = issueId);
  toggleDeleteIssueModal = (issueId: string | null) => (this.isDeleteIssueModalOpen = issueId);
  toggleRelationModal = (issueId: string | null, relationType: TIssueRelationTypes | null) =>
    (this.isRelationModalOpen = { issueId, relationType });
  toggleSubIssuesModal = (issueId: string | null) => (this.isSubIssuesModalOpen = issueId);
  setLastWidgetAction = (widget: TWorkItemWidgets) => {
    this.openWidgets = [widget];
  };
  toggleOpenWidget = (state: TWorkItemWidgets) => {
    if (this.openWidgets && this.openWidgets.includes(state))
      this.openWidgets = this.openWidgets.filter((s) => s !== state);
    else this.openWidgets = [state, ...this.openWidgets];
  };
  setIssueLinkData = (issueLinkData: TIssueLink | null) => (this.issueLinkData = issueLinkData);

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
  // reactions
  addReactions = (issueId: string, reactions: TIssueReaction[]) => this.reaction.addReactions(issueId, reactions);
  fetchReactions = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.reaction.fetchReactions(workspaceSlug, projectId, issueId);
  createReaction = async (workspaceSlug: string, projectId: string, issueId: string, reaction: string) =>
    this.reaction.createReaction(workspaceSlug, projectId, issueId, reaction);
  removeReaction = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    reaction: string,
    userId: string
  ) => this.reaction.removeReaction(workspaceSlug, projectId, issueId, reaction, userId);

  // link
  addLinks = (issueId: string, links: TIssueLink[]) => this.link.addLinks(issueId, links);
  fetchLinks = async (workspaceSlug: string, projectId: string, issueId: string) =>
    this.link.fetchLinks(workspaceSlug, projectId, issueId);
  createLink = async (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssueLink>) =>
    this.link.createLink(workspaceSlug, projectId, issueId, data);
  updateLink = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    linkId: string,
    data: Partial<TIssueLink>
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
  fetchActivities = async (workspaceSlug: string, projectId: string, issueId: string, loaderType?: TActivityLoader) =>
    this.activity.fetchActivities(workspaceSlug, projectId, issueId, loaderType);

  // comment
  fetchComments = async (workspaceSlug: string, projectId: string, issueId: string, loaderType?: TCommentLoader) =>
    this.comment.fetchComments(workspaceSlug, projectId, issueId, loaderType);
  createComment = async (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssueComment>) =>
    this.comment.createComment(workspaceSlug, projectId, issueId, data);
  updateComment = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string,
    data: Partial<TIssueComment>
  ) => this.comment.updateComment(workspaceSlug, projectId, issueId, commentId, data);
  removeComment = async (workspaceSlug: string, projectId: string, issueId: string, commentId: string) =>
    this.comment.removeComment(workspaceSlug, projectId, issueId, commentId);

  // comment reaction
  fetchCommentReactions = async (workspaceSlug: string, projectId: string, commentId: string) =>
    this.commentReaction.fetchCommentReactions(workspaceSlug, projectId, commentId);
  applyCommentReactions = async (commentId: string, commentReactions: TIssueCommentReaction[]) =>
    this.commentReaction.applyCommentReactions(commentId, commentReactions);
  createCommentReaction = async (workspaceSlug: string, projectId: string, commentId: string, reaction: string) =>
    this.commentReaction.createCommentReaction(workspaceSlug, projectId, commentId, reaction);
  removeCommentReaction = async (
    workspaceSlug: string,
    projectId: string,
    commentId: string,
    reaction: string,
    userId: string
  ) => this.commentReaction.removeCommentReaction(workspaceSlug, projectId, commentId, reaction, userId);
}
