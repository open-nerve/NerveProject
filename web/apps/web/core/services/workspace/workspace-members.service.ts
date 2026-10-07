/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, WorkspaceMember, WorkspaceMemberUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A workspace's memberships (M3 design 5.1, 7.3). */
export class WorkspaceMembersService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** Every membership of the workspace, those that ended too (is_active false), in the order they began. */
  async list(slug: string): Promise<WorkspaceMember[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/members", { params: { path: { slug } } })).data;
  }

  /** Changes a member's role; the answer is the membership with its new role. */
  async update(membershipId: string, data: WorkspaceMemberUpdate): Promise<WorkspaceMember> {
    return unwrap(
      await this.api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
        params: { path: { workspace_member_id: membershipId } },
        body: data,
      })
    );
  }

  /** Ends a member's membership, and his memberships of the workspace's projects with it. */
  async remove(membershipId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
        params: { path: { workspace_member_id: membershipId } },
      })
    );
  }
}
