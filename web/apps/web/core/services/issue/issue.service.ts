/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { TLegacyIssueUpdateRequest } from "@nerve/types";
// nerve imports
import type { TIssueParams, TIssueLink, TIssuesResponse, TIssueSubIssues } from "@nerve/types";
// services
import { APIService } from "@/services/api.service";

export class IssueService extends APIService {
  async createIssue(workspaceSlug: string, projectId: string, data: Partial<IssueDetail>): Promise<Issue> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getIssuesFromServer(
    workspaceSlug: string,
    projectId: string,
    queries?: any,
    config = {}
  ): Promise<TIssuesResponse> {
    const path =
      (queries.expand as string)?.includes("issue_relation") && !queries.group_by
        ? `/api/workspaces/${workspaceSlug}/projects/${projectId}/issues-detail/`
        : `/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/`;
    return this.get(
      path,
      {
        params: queries,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getIssues(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.getIssuesFromServer(workspaceSlug, projectId, queries, config);
  }

  async retrieve(workspaceSlug: string, projectId: string, issueId: string, queries?: any): Promise<IssueDetail> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/`, {
      params: queries,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchIssue(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: TLegacyIssueUpdateRequest
  ): Promise<any> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteIssue(workspaceSlug: string, projectId: string, issuesId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issuesId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async subIssues(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>
  ): Promise<TIssueSubIssues> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/sub-issues/`, {
      params: queries,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async addSubIssues(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: { sub_issue_ids: string[] }
  ): Promise<TIssueSubIssues> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/sub-issues/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async fetchIssueLinks(workspaceSlug: string, projectId: string, issueId: string): Promise<TIssueLink[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/issue-links/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async createIssueLink(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Partial<TIssueLink>
  ): Promise<TIssueLink> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/issue-links/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async updateIssueLink(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    linkId: string,
    data: Partial<TIssueLink>
  ): Promise<TIssueLink> {
    return this.patch(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/issue-links/${linkId}/`,
      data
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async deleteIssueLink(workspaceSlug: string, projectId: string, issueId: string, linkId: string): Promise<any> {
    return this.delete(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/issue-links/${linkId}/`
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getIssueMetaFromURL(
    workspaceSlug: string,
    projectId: string,
    issueId: string
  ): Promise<{
    project_identifier: string;
    sequence_id: string;
  }> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/meta/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async retrieveWithIdentifier(
    workspaceSlug: string,
    project_identifier: string,
    issue_sequence: string,
    queries?: any
  ): Promise<IssueDetail> {
    return this.get(`/api/workspaces/${workspaceSlug}/work-items/${project_identifier}-${issue_sequence}/`, {
      params: queries,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
