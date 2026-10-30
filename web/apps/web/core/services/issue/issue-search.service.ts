/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueCollection, operations } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export type IssueSearchQuery = NonNullable<operations["searchIssues"]["parameters"]["query"]>;

export class IssueSearchService {
  constructor(private readonly api: ApiClient) {}

  async search(slug: string, query: IssueSearchQuery): Promise<IssueCollection> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/issue-search", {
        params: { path: { slug }, query },
      })
    );
  }
}
