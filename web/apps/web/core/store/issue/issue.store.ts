/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { set, update } from "lodash-es";
import { action, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// types
import type { Issue, IssueDetail } from "@nerve/api-client";
// helpers
import { getCurrentDateTimeInISO } from "@nerve/utils";
import { rootStore } from "@/lib/store-context";
export type IIssueStore = {
  // observables
  issuesMap: Record<string, Issue | IssueDetail>; // Record defines issue_id as key and Issue or IssueDetail as value
  issuesIdentifierMap: Record<string, string>; // Record defines issue_identifier as key and issue_id as value
  // actions
  addIssue(issues: (Issue | IssueDetail)[]): void;
  addIssueIdentifier(issueIdentifier: string, issueId: string): void;
  updateIssue(issueId: string, issue: Partial<IssueDetail>): void;
  removeIssue(issueId: string): void;
  // helper methods
  getIssueById(issueId: string): Issue | IssueDetail | undefined;
  getIssueDetailById(issueId: string): IssueDetail | undefined;
  getIssueIdByIdentifier(issueIdentifier: string): undefined | string;
  getIssuesByIds(issueIds: string[], type: "archived" | "un-archived"): (Issue | IssueDetail)[]; // Record defines issue_id as key and Issue or IssueDetail as value
};

export class IssueStore implements IIssueStore {
  // observables
  issuesMap: Record<string, Issue | IssueDetail> = {};
  issuesIdentifierMap: { [issue_identifier: string]: string } = {};

  constructor() {
    makeObservable(this, {
      // observable
      issuesMap: observable,
      issuesIdentifierMap: observable,
      // actions
      addIssue: action,
      addIssueIdentifier: action,
      updateIssue: action,
      removeIssue: action,
    });
  }

  // actions
  /**
   * @description This method will add issues to the issuesMap
   * @param {(Issue | IssueDetail)[]} issues
   * @returns {void}
   */
  addIssue = (issues: (Issue | IssueDetail)[]) => {
    if (issues && issues.length <= 0) return;
    runInAction(() => {
      issues.forEach((issue) => {
        // add issue identifier to the issuesIdentifierMap
        const projectIdentifier = rootStore.projectRoot.project.getProjectIdentifierById(issue?.project_id);
        const workItemSequenceId = issue?.sequence_id;
        const issueIdentifier = `${projectIdentifier}-${workItemSequenceId}`;
        set(this.issuesIdentifierMap, issueIdentifier, issue.id);

        if (!this.issuesMap[issue.id]) set(this.issuesMap, issue.id, issue);
        else update(this.issuesMap, issue.id, (prevIssue) => ({ ...prevIssue, ...issue }));
      });
    });
  };

  /**
   * @description This method will add issue_identifier to the issuesIdentifierMap
   * @param issueIdentifier
   * @param issueId
   * @returns {void}
   */
  addIssueIdentifier = (issueIdentifier: string, issueId: string) => {
    if (!issueIdentifier || !issueId) return;
    runInAction(() => {
      set(this.issuesIdentifierMap, issueIdentifier, issueId);
    });
  };

  /**
   * @description This method will update the issue in the issuesMap
   * @param {string} issueId
   * @param {Partial<IssueDetail>} issue
   * @returns {void}
   */
  updateIssue = (issueId: string, issue: Partial<IssueDetail>) => {
    if (!issue || !issueId || !this.issuesMap[issueId]) return;
    runInAction(() => {
      set(this.issuesMap, [issueId, "updated_at"], getCurrentDateTimeInISO());
      Object.keys(issue).forEach((key) => {
        set(this.issuesMap, [issueId, key], issue[key as keyof IssueDetail]);
      });
    });
  };

  /**
   * @description This method will remove the issue from the issuesMap
   * @param {string} issueId
   * @returns {void}
   */
  removeIssue = (issueId: string) => {
    if (!issueId || !this.issuesMap[issueId]) return;
    runInAction(() => {
      delete this.issuesMap[issueId];
    });
  };

  // helper methods
  /**
   * @description This method will return the issue from the issuesMap
   * @param {string} issueId
   * @returns {Issue | IssueDetail | undefined}
   */
  getIssueById = computedFn((issueId: string): Issue | IssueDetail | undefined => {
    if (!issueId || !this.issuesMap[issueId]) return undefined;
    return this.issuesMap[issueId];
  });

  getIssueDetailById = computedFn((issueId: string): IssueDetail | undefined => {
    const issue = this.getIssueById(issueId);
    return issue && "description_html" in issue ? issue : undefined;
  });

  /**
   * @description This method will return the issue_id from the issuesIdentifierMap
   * @param {string} issueIdentifier
   * @returns {string | undefined}
   */
  getIssueIdByIdentifier = computedFn((issueIdentifier: string) => {
    if (!issueIdentifier || !this.issuesIdentifierMap[issueIdentifier]) return undefined;
    return this.issuesIdentifierMap[issueIdentifier];
  });

  /**
   * @description This method will return the issues from the issuesMap
   * @param {string[]} issueIds
   * @param {boolean} archivedIssues
   * @returns {Record<string, Issue | IssueDetail> | undefined}
   */
  getIssuesByIds = computedFn((issueIds: string[], type: "archived" | "un-archived") => {
    if (!issueIds || issueIds.length <= 0) return [];
    const filteredIssues: (Issue | IssueDetail)[] = [];
    Object.values(issueIds).forEach((issueId) => {
      // if type is archived then check archived_at is not null
      // if type is un-archived then check archived_at is null
      const issue = this.issuesMap[issueId];
      if (issue && ((type === "archived" && issue.archived_at) || (type === "un-archived" && !issue?.archived_at))) {
        filteredIssues.push(issue);
      }
    });
    return filteredIssues;
  });
}
