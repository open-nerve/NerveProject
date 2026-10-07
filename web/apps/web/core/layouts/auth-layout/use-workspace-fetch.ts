/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * What WorkspaceAuthWrapper shows for the address's workspace (M3 design 8.3): that nerve cannot be reached, with a
 * retry; a wait for the caller's workspaces; that the workspace is not found, whether it does not exist or he is not
 * a member of it (and whether he has others); or its pages.
 */
export type WorkspaceAccess =
  | { kind: "unavailable"; retry: () => void }
  | { kind: "loading" }
  | { kind: "not-found"; hasWorkspaces: boolean }
  | { kind: "ready"; workspace: Workspace };

/**
 * The workspace side of what a page of a workspace fetches as it mounts (M3 design 3.1, 7.1), and the one decision
 * whether the address's workspace is the caller's: his workspaces, which decide it; once his list has it, its members
 * and his navigation settings in it. Gives what the wrapper shows.
 */
export function useWorkspaceFetch(workspaceSlug: string | undefined): WorkspaceAccess {
  const {
    workspaces,
    fetchWorkspaces,
    getWorkspaceBySlug,
    preferences: { fetchPreferences },
  } = useWorkspace();
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces());
  // the address's workspace is the caller's once his list has it
  const workspace = workspaceSlug === undefined ? null : getWorkspaceBySlug(workspaceSlug);
  useWorkspaceMembersFetch(workspace);
  useSessionSWR(workspace && ["WORKSPACE_PREFERENCES", workspace.id, workspace.slug], (id, slug) =>
    fetchPreferences({ id, slug })
  );
  if (listed.error) return { kind: "unavailable", retry: () => void listed.mutate() };
  if (workspaces === undefined) return { kind: "loading" };
  if (workspace === null) return { kind: "not-found", hasWorkspaces: workspaces.length > 0 };
  return { kind: "ready", workspace };
}
