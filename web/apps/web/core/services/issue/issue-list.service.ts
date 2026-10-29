/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueListPage, operations } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export type ProjectIssueQuery = NonNullable<operations["listIssues"]["parameters"]["query"]>;
export type MemberIssueQuery = operations["listMemberIssues"]["parameters"]["query"];

/** Project and member pages as nerve answers them, using the generated client's query serialization. */
export class IssueListService {
  /** The client belongs to the RootStore's session. */
  constructor(private readonly api: ApiClient) {}

  async listProject(projectId: string, query: ProjectIssueQuery): Promise<IssueListPage> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/issues", {
        params: { path: { project_id: projectId }, query },
      })
    );
  }

  async listMember(slug: string, userId: string, query: MemberIssueQuery): Promise<IssueListPage> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/members/{user_id}/issues", {
        params: { path: { slug, user_id: userId }, query },
      })
    );
  }
}
