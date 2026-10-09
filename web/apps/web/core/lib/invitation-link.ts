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

/**
 * The sign-in page ("/") or the sign-up page for an invitation's link (M3 design 7.4): its header names the
 * invitation's workspace, and once signed in, the account comes back to the link (next_path, M2 design 3.18).
 */
export function invitationAuthPath(
  path: "/" | "/sign-up",
  invitation: Pick<WorkspaceInvitation, "id" | "token">
): string {
  const query = new URLSearchParams({
    invitation_id: invitation.id,
    token: invitation.token,
    next_path: invitationLink("", invitation),
  });
  return `${path}?${query.toString()}`;
}
