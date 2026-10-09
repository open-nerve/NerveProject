/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";

/**
 * The link of an invitation (M3 design 7.4): the invitation page at origin, with the invitation's id and the token
 * nerve gives an admin with it, each a value of the query, encoded as one.
 */
export function invitationLink(origin: string, invitation: Pick<WorkspaceInvitation, "id" | "token">): string {
  const query = new URLSearchParams({ invitation_id: invitation.id, token: invitation.token });
  return `${origin}/workspace-invitations?${query.toString()}`;
}
