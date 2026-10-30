/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, DescriptionVersionPage, DescriptionVersionDetail, operations } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
export class WorkItemVersionService {
  constructor(private readonly api: ApiClient) {}
  async listDescriptionVersions(
    issueId: string,
    query?: operations["listDescriptionVersions"]["parameters"]["query"]
  ): Promise<DescriptionVersionPage> {
    return unwrap(
      await this.api.GET("/api/v0/issues/{issue_id}/description-versions", {
        params: { path: { issue_id: issueId }, query },
      })
    );
  }
  async retrieveDescriptionVersion(versionId: string): Promise<DescriptionVersionDetail> {
    return unwrap(
      await this.api.GET("/api/v0/issue-description-versions/{version_id}", {
        params: { path: { version_id: versionId } },
      })
    );
  }
}
