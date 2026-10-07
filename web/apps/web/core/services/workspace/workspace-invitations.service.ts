/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  WorkspaceInvitation,
  WorkspaceInvitationUpdate,
  WorkspaceInvitationsCreate,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A workspace's invitations, as its admins manage them (M3 design 5.1, 7.3). */
export class WorkspaceInvitationsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The invitations, pending and declined, newest first, each with the token of its link. */
  async list(slug: string): Promise<WorkspaceInvitation[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } } })).data;
  }

  /** Invites the addresses data lists, all of them or none; the answer is the new invitations, in data's order. */
  async create(slug: string, data: WorkspaceInvitationsCreate): Promise<WorkspaceInvitation[]> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } }, body: data })
    ).data;
  }

  /** Changes an invitation's role; the answer is the invitation as nerve now holds it. */
  async update(invitationId: string, data: WorkspaceInvitationUpdate): Promise<WorkspaceInvitation> {
    return unwrap(
      await this.api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: { invitation_id: invitationId } },
        body: data,
      })
    );
  }

  /** Deletes an invitation: its link stops working. */
  async delete(invitationId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: { invitation_id: invitationId } },
      })
    );
  }
}
