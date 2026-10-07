/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, InvitationPreview } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * An invitation as its link shows it to whoever holds the link (M3 design 5.1, 7.4): public, the same with a session
 * or without one, so the web app asks with publicClient.
 */
export async function previewInvitation(
  client: ApiClient,
  invitationId: string,
  token: string
): Promise<InvitationPreview> {
  return unwrap(
    await client.GET("/api/v0/workspace-invitations/{invitation_id}", {
      params: { path: { invitation_id: invitationId }, query: { token } },
    })
  );
}
