/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { EUserWorkspaceRoles } from "@nerve/types";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The fetches of a workspace's members settings (M3 design 7.1: a page fetches what its caller may read): the
 * members, as every page of the workspace has them (useWorkspaceMembersFetch); the invitations for an admin alone, as
 * nerve shows them to no one else. The caller's role is the one his workspaces list gives (Workspace.role).
 */
export function useMembersSettingsFetch(workspaceSlug: string | undefined): void {
  const { getWorkspaceBySlug } = useWorkspace();
  const {
    workspace: { fetchWorkspaceMemberInvitations },
  } = useMember();
  const workspace = workspaceSlug === undefined ? null : getWorkspaceBySlug(workspaceSlug);
  useWorkspaceMembersFetch(workspace);
  useSessionSWR(
    workspace?.role === EUserWorkspaceRoles.ADMIN ? ["WORKSPACE_INVITATIONS", workspace.id, workspace.slug] : null,
    (id, slug) => fetchWorkspaceMemberInvitations({ id, slug })
  );
}
