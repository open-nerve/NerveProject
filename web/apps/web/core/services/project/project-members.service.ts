/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ProjectMember, ProjectMembersAdd, ProjectMemberUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A project's memberships (M3 design 5.1, 7.3); the caller's own joining and leaving are the projects service's. */
export class ProjectMembersService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's active memberships, in the order they began. */
  async list(projectId: string): Promise<ProjectMember[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/members", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** Adds members of the workspace to the project, all of them or none; the answer is their memberships. */
  async add(projectId: string, data: ProjectMembersAdd): Promise<ProjectMember[]> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    ).data;
  }

  /** Changes a member's role; the answer is the membership with its new role. */
  async update(membershipId: string, data: ProjectMemberUpdate): Promise<ProjectMember> {
    return unwrap(
      await this.api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membershipId } },
        body: data,
      })
    );
  }

  /** Ends a member's membership of the project. */
  async remove(membershipId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membershipId } },
      })
    );
  }
}
