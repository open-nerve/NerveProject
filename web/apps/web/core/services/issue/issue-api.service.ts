/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueCreate, IssueDetail, IssueUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** Core issue operations: only confirmed server answers, with no local entity or response conversion. */
export class IssueApiService {
  /** The client belongs to the RootStore's session. */
  constructor(private readonly api: ApiClient) {}

  async get(issueId: string): Promise<IssueDetail> {
    return unwrap(await this.api.GET("/api/v0/issues/{issue_id}", { params: { path: { issue_id: issueId } } }));
  }

  async create(projectId: string, body: IssueCreate): Promise<IssueDetail> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/issues", {
        params: { path: { project_id: projectId } },
        body,
      })
    );
  }

  async update(issueId: string, body: IssueUpdate): Promise<IssueDetail> {
    return unwrap(
      await this.api.PATCH("/api/v0/issues/{issue_id}", {
        params: { path: { issue_id: issueId } },
        body,
      })
    );
  }

  async delete(issueId: string): Promise<void> {
    return unwrap(await this.api.DELETE("/api/v0/issues/{issue_id}", { params: { path: { issue_id: issueId } } }));
  }

  async archive(issueId: string): Promise<IssueDetail> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/archive", { params: { path: { issue_id: issueId } } })
    );
  }

  async unarchive(issueId: string): Promise<IssueDetail> {
    return unwrap(
      await this.api.POST("/api/v0/issues/{issue_id}/unarchive", { params: { path: { issue_id: issueId } } })
    );
  }
}
