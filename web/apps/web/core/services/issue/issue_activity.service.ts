/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueActivityPage, operations } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
export class IssueActivityService {
  constructor(private readonly api: ApiClient) {}
  async getIssueActivities(
    issueId: string,
    query?: operations["listIssueActivities"]["parameters"]["query"]
  ): Promise<IssueActivityPage> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/activities", { params: { path: { issue_id: issueId }, query } })
    );
  }
}
