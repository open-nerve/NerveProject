/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { uniq, get, set } from "lodash-es";
import { action, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { TIssueRelationIdMap, TIssueRelationMap, TIssueRelation, TIssueRelationTypes } from "@nerve/types";
// components
import type { TRelationObject } from "@/components/issues/issue-detail-widgets/relations";
import { REVERSE_RELATIONS } from "@nerve/constants";
// services
import { IssueRelationService } from "@/services/issue";
// types
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IIssueDetail } from "./root.store";
export interface IIssueRelationStoreActions {
  // actions
  fetchRelations: (workspaceSlug: string, projectId: string, issueId: string) => Promise<TIssueRelation>;
  createRelation: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    issues: string[]
  ) => Promise<Issue[]>;
  removeRelation: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    related_issue: string,
    updateLocally?: boolean
  ) => Promise<void>;
}

export interface IIssueRelationStore extends IIssueRelationStoreActions {
  // observables
  relationMap: TIssueRelationMap;
  // helper methods
  getRelationsByIssueId: (issueId: string) => TIssueRelationIdMap | undefined;
  getRelationCountByIssueId: (
    issueId: string,
    ISSUE_RELATION_OPTIONS: { [key in TIssueRelationTypes]?: TRelationObject }
  ) => number;
}

export class IssueRelationStore implements IIssueRelationStore {
  // observables
  relationMap: TIssueRelationMap = {};
  // root store
  rootIssueDetailStore: IIssueDetail;
  // services
  issueRelationService;

  constructor(rootStore: IIssueDetail) {
    makeObservable(this, {
      // observables
      relationMap: observable,
      // actions
      fetchRelations: action,
      createRelation: action,
      removeRelation: action,
    });
    // root store
    this.rootIssueDetailStore = rootStore;
    // services
    this.issueRelationService = new IssueRelationService();
  }

  // // helper methods
  getRelationsByIssueId = (issueId: string) => {
    if (!issueId) return undefined;
    return this.relationMap?.[issueId] ?? undefined;
  };

  getRelationCountByIssueId = computedFn(
    (issueId: string, ISSUE_RELATION_OPTIONS: { [key in TIssueRelationTypes]?: TRelationObject }) => {
      const issueRelations = this.getRelationsByIssueId(issueId);

      const issueRelationKeys = (Object.keys(issueRelations ?? {}) as TIssueRelationTypes[]).filter(
        (relationKey) => !!ISSUE_RELATION_OPTIONS[relationKey]
      );

      return issueRelationKeys.reduce((acc, curr) => acc + (issueRelations?.[curr]?.length ?? 0), 0);
    }
  );

  // actions
  fetchRelations = async (workspaceSlug: string, projectId: string, issueId: string) => {
    const root = this.rootIssueDetailStore.rootIssueStore;
    const token = root.captureRead();
    const response = await this.issueRelationService.listIssueRelations(workspaceSlug, projectId, issueId);

    const projects = uniq([
      projectId,
      ...Object.values(response)
        .flat()
        .map((issue) => issue.project_id)
        .filter(Boolean),
    ]);
    if (!root.isReadCurrent(token, projects)) throw new SessionChangedError();
    runInAction(() => {
      Object.keys(response).forEach((key) => {
        const relation_key = key as TIssueRelationTypes;
        const relation_issues = response[relation_key];
        const issues = relation_issues.flat().map((issue) => issue);
        if (issues && issues.length > 0) root.acceptListIssues(issues, token);
        set(
          this.relationMap,
          [issueId, relation_key],
          issues && issues.length > 0 ? issues.map((issue) => issue.id) : []
        );
      });
    });

    return response;
  };

  createRelation = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    issues: string[]
  ) => {
    const root = this.rootIssueDetailStore.rootIssueStore;
    const token = root.captureRead();
    const response = await this.issueRelationService.createIssueRelations(workspaceSlug, projectId, issueId, {
      relation_type: relationType,
      issues,
    });

    const projects = uniq([projectId, ...response.map((issue) => issue.project_id).filter(Boolean)]);
    if (!root.isReadCurrent(token, projects)) throw new SessionChangedError();
    const reverseRelatedType = REVERSE_RELATIONS[relationType];

    const issuesOfRelation = get(this.relationMap, [issueId, relationType]) ?? [];

    if (response && response.length > 0)
      runInAction(() => {
        response.forEach((issue) => {
          const issuesOfRelated = get(this.relationMap, [issue.id, reverseRelatedType]);
          root.issues.acceptRead(issue, token);
          issuesOfRelation.push(issue.id);

          if (!issuesOfRelated) {
            set(this.relationMap, [issue.id, reverseRelatedType], [issueId]);
          } else {
            set(this.relationMap, [issue.id, reverseRelatedType], uniq([...issuesOfRelated, issueId]));
          }
        });
        set(this.relationMap, [issueId, relationType], uniq(issuesOfRelation));
      });

    // fetching activity
    this.rootIssueDetailStore.activity.fetchActivities(workspaceSlug, projectId, issueId);
    return response;
  };

  removeRelation = async (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    relationType: TIssueRelationTypes,
    related_issue: string,
    updateLocally = false
  ) => {
    try {
      const relationIndex = this.relationMap[issueId]?.[relationType]?.findIndex(
        (_issueId) => _issueId === related_issue
      );
      if (relationIndex >= 0)
        runInAction(() => {
          this.relationMap[issueId]?.[relationType]?.splice(relationIndex, 1);
        });

      if (!updateLocally) {
        await this.issueRelationService.deleteIssueRelation(workspaceSlug, projectId, issueId, {
          relation_type: relationType,
          related_issue,
        });
      }

      // While removing one relation, reverse of the relation should also be removed
      const reverseRelatedType = REVERSE_RELATIONS[relationType];
      const relatedIndex = this.relationMap[related_issue]?.[reverseRelatedType]?.findIndex(
        (_issueId) => _issueId === related_issue
      );
      if (relationIndex >= 0)
        runInAction(() => {
          this.relationMap[related_issue]?.[reverseRelatedType]?.splice(relatedIndex, 1);
        });

      // fetching activity
      this.rootIssueDetailStore.activity.fetchActivities(workspaceSlug, projectId, issueId);
    } catch (error) {
      this.fetchRelations(workspaceSlug, projectId, issueId);
      throw error;
    }
  };
}
