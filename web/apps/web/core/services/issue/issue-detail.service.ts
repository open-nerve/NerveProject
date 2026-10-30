/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type {
  ApiClient,
  IssueCollection,
  IssueLink,
  IssueLinkCollection,
  IssueLinkCreate,
  IssueLinkPatch,
  IssueRelationCollection,
  IssueRelationsCreate,
  SubIssuesAdd,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** Native detail resources; state, ordering and confirmation belong to the stores. */
export class IssueDetailService {
  constructor(private readonly api: ApiClient) {}

  async listSubIssues(issueId: string): Promise<IssueCollection> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/sub-issues", { params: { path: { issue_id: issueId } } })
    );
  }

  async addSubIssues(issueId: string, body: SubIssuesAdd): Promise<IssueCollection> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/sub-issues", { params: { path: { issue_id: issueId } }, body })
    );
  }

  async listRelations(issueId: string): Promise<IssueRelationCollection> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/relations", { params: { path: { issue_id: issueId } } })
    );
  }

  async createRelations(issueId: string, body: IssueRelationsCreate): Promise<IssueRelationCollection> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/relations", { params: { path: { issue_id: issueId } }, body })
    );
  }

  async deleteRelation(relationId: string): Promise<void> {
    return unwrap(
      await this.api.DELETE("/api/v0/issue-relations/{relation_id}", { params: { path: { relation_id: relationId } } })
    );
  }

  async listLinks(issueId: string): Promise<IssueLinkCollection> {
    return unwrap(await this.api.GET("/api/v0/issues/{issue_id}/links", { params: { path: { issue_id: issueId } } }));
  }

  async createLink(issueId: string, body: IssueLinkCreate): Promise<IssueLink> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/links", { params: { path: { issue_id: issueId } }, body })
    );
  }

  async updateLink(linkId: string, body: IssueLinkPatch): Promise<IssueLink> {
    return unwrap(
      await this.api.PATCH("/api/v0/issue-links/{link_id}", { params: { path: { link_id: linkId } }, body })
    );
  }

  async deleteLink(linkId: string): Promise<void> {
    return unwrap(await this.api.DELETE("/api/v0/issue-links/{link_id}", { params: { path: { link_id: linkId } } }));
  }
}
