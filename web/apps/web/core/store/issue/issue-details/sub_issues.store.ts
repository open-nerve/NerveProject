/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import { pull, concat, uniq, set, update } from "lodash-es";
import { action, makeObservable, observable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// Nerve Imports
import type {
  TIssueSubIssues,
  TIssueSubIssuesStateDistributionMap,
  TIssueSubIssuesIdMap,
  TSubIssuesStateDistribution,
} from "@nerve/types";
// services
import { IssueService } from "@/services/issue";
// store
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IIssueDetail } from "./root.store";
import type { IWorkItemSubIssueFiltersStore } from "./sub_issues_filter.store";
import { WorkItemSubIssueFiltersStore } from "./sub_issues_filter.store";

export interface IIssueSubIssuesStoreActions {
  fetchSubIssues: (workspaceSlug: string, projectId: string, parentIssueId: string) => Promise<TIssueSubIssues>;
  createSubIssues: (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issueIds: string[]
  ) => Promise<void>;
  updateSubIssue: (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issueId: string,
    issueData: Partial<IssueDetail>,
    oldIssue?: Partial<IssueDetail>,
    fromModal?: boolean
  ) => Promise<void>;
  removeSubIssue: (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => Promise<void>;
  deleteSubIssue: (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => Promise<void>;
}

type TSubIssueHelpersKeys = "issue_visibility" | "preview_loader" | "issue_loader";
type TSubIssueHelpers = Record<TSubIssueHelpersKeys, string[]>;
export interface IIssueSubIssuesStore extends IIssueSubIssuesStoreActions {
  // observables
  subIssuesStateDistribution: TIssueSubIssuesStateDistributionMap;
  subIssues: TIssueSubIssuesIdMap;
  subIssueHelpers: Record<string, TSubIssueHelpers>; // parent_issue_id -> TSubIssueHelpers
  filters: IWorkItemSubIssueFiltersStore;
  // helper methods
  stateDistributionByIssueId: (issueId: string) => TSubIssuesStateDistribution | undefined;
  subIssuesByIssueId: (issueId: string) => string[] | undefined;
  subIssueHelpersByIssueId: (issueId: string) => TSubIssueHelpers;
  // actions
  fetchOtherProjectProperties: (projectIds: string[]) => Promise<void>;
  setSubIssueHelpers: (parentIssueId: string, key: TSubIssueHelpersKeys, value: string) => void;
}

export class IssueSubIssuesStore implements IIssueSubIssuesStore {
  // observables
  subIssuesStateDistribution: TIssueSubIssuesStateDistributionMap = {};
  subIssues: TIssueSubIssuesIdMap = {};
  subIssueHelpers: Record<string, TSubIssueHelpers> = {};

  filters: IWorkItemSubIssueFiltersStore;
  // root store
  rootIssueDetailStore: IIssueDetail;
  // services
  issueService;

  constructor(rootStore: IIssueDetail) {
    makeObservable(this, {
      // observables
      subIssuesStateDistribution: observable,
      subIssues: observable,
      subIssueHelpers: observable,
      // actions
      setSubIssueHelpers: action,
      fetchSubIssues: action,
      createSubIssues: action,
      updateSubIssue: action,
      removeSubIssue: action,
      deleteSubIssue: action,
      fetchOtherProjectProperties: action,
    });
    this.filters = new WorkItemSubIssueFiltersStore(this);
    // root store
    this.rootIssueDetailStore = rootStore;
    // services
    this.issueService = new IssueService();
  }

  // helper methods
  stateDistributionByIssueId = (issueId: string) => {
    if (!issueId) return undefined;
    return this.subIssuesStateDistribution[issueId] ?? undefined;
  };

  subIssuesByIssueId = computedFn((issueId: string) => this.subIssues[issueId]);

  subIssueHelpersByIssueId = (issueId: string) => ({
    preview_loader: this.subIssueHelpers?.[issueId]?.preview_loader || [],
    issue_visibility: this.subIssueHelpers?.[issueId]?.issue_visibility || [],
    issue_loader: this.subIssueHelpers?.[issueId]?.issue_loader || [],
  });

  // actions
  setSubIssueHelpers = (parentIssueId: string, key: TSubIssueHelpersKeys, value: string) => {
    if (!parentIssueId || !key || !value) return;

    update(this.subIssueHelpers, [parentIssueId, key], (_subIssueHelpers: string[] = []) => {
      if (_subIssueHelpers.includes(value)) return pull(_subIssueHelpers, value);
      return concat(_subIssueHelpers, value);
    });
  };

  fetchSubIssues = async (workspaceSlug: string, projectId: string, parentIssueId: string) => {
    const root = this.rootIssueDetailStore.rootIssueStore;
    const token = root.captureRead();
    const response = await this.issueService.subIssues(workspaceSlug, projectId, parentIssueId);

    if (!root.isReadCurrent(token, [projectId])) throw new SessionChangedError();
    const subIssuesStateDistribution = response?.state_distribution ?? {};

    const issueList = (response.sub_issues ?? []) as Issue[];

    root.acceptListIssues(issueList, token);

    // fetch other issues states and members when sub-issues are from different project
    if (issueList && issueList.length > 0) {
      const otherProjectIds = uniq(
        issueList.map((issue) => issue.project_id).filter((id) => !!id && id !== projectId)
      ) as string[];
      this.fetchOtherProjectProperties(otherProjectIds);
    }
    await root.issues.refreshAuthoritative(parentIssueId);

    runInAction(() => {
      set(this.subIssuesStateDistribution, parentIssueId, subIssuesStateDistribution);
      set(
        this.subIssues,
        parentIssueId,
        issueList.map((issue) => issue.id)
      );
    });

    return response;
  };

  createSubIssues = async (workspaceSlug: string, projectId: string, parentIssueId: string, issueIds: string[]) => {
    const root = this.rootIssueDetailStore.rootIssueStore;
    const token = root.captureRead();
    const response = await this.issueService.addSubIssues(workspaceSlug, projectId, parentIssueId, {
      sub_issue_ids: issueIds,
    });

    if (!root.isReadCurrent(token, [projectId])) throw new SessionChangedError();
    const subIssuesStateDistribution = response?.state_distribution ?? {};
    const subIssues = Array.isArray(response?.sub_issues) ? response.sub_issues : [];

    // fetch other issues states and members when sub-issues are from different project
    if (subIssues && subIssues.length > 0) {
      const otherProjectIds = uniq(
        subIssues.map((issue) => issue.project_id).filter((id) => !!id && id !== projectId)
      ) as string[];
      this.fetchOtherProjectProperties(otherProjectIds);
    }

    runInAction(() => {
      Object.keys(subIssuesStateDistribution).forEach((key) => {
        const stateGroup = key as keyof TSubIssuesStateDistribution;
        update(this.subIssuesStateDistribution, [parentIssueId, stateGroup], (stateDistribution) => {
          if (!stateDistribution) return subIssuesStateDistribution[stateGroup];
          return concat(stateDistribution, subIssuesStateDistribution[stateGroup]);
        });
      });

      const subIssueIds = subIssues.map((issue) => issue.id);
      update(this.subIssues, [parentIssueId], (issues) => {
        if (!issues) return subIssueIds;
        return concat(issues, subIssueIds);
      });
    });

    root.acceptListIssues(subIssues, token);
    await root.issues.refreshAuthoritative(parentIssueId);

    return;
  };

  updateSubIssue = async (
    workspaceSlug: string,
    projectId: string,
    parentIssueId: string,
    issueId: string,
    issueData: Partial<IssueDetail>,
    oldIssue: Partial<IssueDetail> = {},
    fromModal: boolean = false
  ) => {
    if (!fromModal)
      await this.rootIssueDetailStore.rootIssueStore.projectIssues.updateIssue(
        workspaceSlug,
        projectId,
        issueId,
        issueData
      );

    // parent update
    if (issueData.hasOwnProperty("parent_id") && issueData.parent_id !== oldIssue.parent_id) {
      runInAction(() => {
        if (oldIssue.parent_id) pull(this.subIssues[oldIssue.parent_id], issueId);
        if (issueData.parent_id)
          set(this.subIssues, [issueData.parent_id], concat(this.subIssues[issueData.parent_id], issueId));
      });
    }

    // state update
    if (issueData.hasOwnProperty("state_id") && issueData.state_id !== oldIssue.state_id) {
      let oldIssueStateGroup: string | undefined = undefined;
      let issueStateGroup: string | undefined = undefined;

      if (oldIssue.state_id) {
        const state = this.rootIssueDetailStore.rootIssueStore.rootStore.state.getStateById(oldIssue.state_id);
        if (state?.group) oldIssueStateGroup = state.group;
      }

      if (issueData.state_id) {
        const state = this.rootIssueDetailStore.rootIssueStore.rootStore.state.getStateById(issueData.state_id);
        if (state?.group) issueStateGroup = state.group;
      }

      if (oldIssueStateGroup && issueStateGroup && issueStateGroup !== oldIssueStateGroup) {
        runInAction(() => {
          if (oldIssueStateGroup)
            update(this.subIssuesStateDistribution, [parentIssueId, oldIssueStateGroup], (stateDistribution) => {
              if (!stateDistribution) return;
              return pull(stateDistribution, issueId);
            });

          if (issueStateGroup)
            update(this.subIssuesStateDistribution, [parentIssueId, issueStateGroup], (stateDistribution) => {
              if (!stateDistribution) return [issueId];
              return concat(stateDistribution, issueId);
            });
        });
      }
    }

    return;
  };

  removeSubIssue = async (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => {
    await this.rootIssueDetailStore.rootIssueStore.projectIssues.updateIssue(workspaceSlug, projectId, issueId, {
      parent_id: null,
    });

    const issue = this.rootIssueDetailStore.issue.getIssueById(issueId);
    if (issue && issue.state_id) {
      let issueStateGroup: string | undefined = undefined;
      const state = this.rootIssueDetailStore.rootIssueStore.rootStore.state.getStateById(issue.state_id);
      if (state?.group) issueStateGroup = state.group;

      if (issueStateGroup) {
        runInAction(() => {
          if (issueStateGroup)
            update(this.subIssuesStateDistribution, [parentIssueId, issueStateGroup], (stateDistribution) => {
              if (!stateDistribution) return;
              return pull(stateDistribution, issueId);
            });
        });
      }
    }

    runInAction(() => {
      pull(this.subIssues[parentIssueId], issueId);
    });
    await this.rootIssueDetailStore.rootIssueStore.issues.refreshAuthoritative(parentIssueId);
    return;
  };

  deleteSubIssue = async (workspaceSlug: string, projectId: string, parentIssueId: string, issueId: string) => {
    await this.rootIssueDetailStore.rootIssueStore.projectIssues.removeIssue(workspaceSlug, projectId, issueId);

    const issue = this.rootIssueDetailStore.issue.getIssueById(issueId);
    if (issue && issue.state_id) {
      let issueStateGroup: string | undefined = undefined;
      const state = this.rootIssueDetailStore.rootIssueStore.rootStore.state.getStateById(issue.state_id);
      if (state?.group) issueStateGroup = state.group;

      if (issueStateGroup) {
        runInAction(() => {
          if (issueStateGroup)
            update(this.subIssuesStateDistribution, [parentIssueId, issueStateGroup], (stateDistribution) => {
              if (!stateDistribution) return;
              return pull(stateDistribution, issueId);
            });
        });
      }
    }

    runInAction(() => {
      pull(this.subIssues[parentIssueId], issueId);
    });
    await this.rootIssueDetailStore.rootIssueStore.issues.refreshAuthoritative(parentIssueId);
    return;
  };

  fetchOtherProjectProperties = async (projectIds: string[]) => {
    if (projectIds.length > 0) {
      for (const projectId of projectIds) {
        // fetching other project states
        this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);
        // fetching other project members
        this.rootIssueDetailStore.rootIssueStore.rootStore.memberRoot.project.fetchProjectMembers(projectId);
        // fetching other project labels
        this.rootIssueDetailStore.rootIssueStore.rootStore.label.fetchProjectLabels(projectId);
      }
    }
  };
}
