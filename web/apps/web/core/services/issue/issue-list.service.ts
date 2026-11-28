/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient, IssueListPage, operations } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

export type ProjectIssueQuery = NonNullable<operations["listIssues"]["parameters"]["query"]>;
export type MemberIssueQuery = operations["listMemberIssues"]["parameters"]["query"];
/**
 * The workspace list's query (M7/P6b spec 2.3): nerve's, which has no grouping, its list being flat. The two grouping
 * fields are typed `never` so that the list engine's readers of a query's grouping read none on it, unchanged.
 */
export type WorkspaceIssueQuery = NonNullable<operations["listWorkspaceIssues"]["parameters"]["query"]> & {
  group_by?: never;
  sub_group_by?: never;
};

/** Project, member and workspace pages as nerve answers them, using the generated client's query serialization. */
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

  async listWorkspace(slug: string, query: WorkspaceIssueQuery): Promise<IssueListPage> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/issues", { params: { path: { slug }, query } }));
  }
}
