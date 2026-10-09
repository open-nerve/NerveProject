/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
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
 * whether the address's workspace is the caller's: his workspaces, which decide it; once his list has it, its members,
 * his navigation settings in it, its projects that are not archived and the states of those he is a member of. Gives
 * what the wrapper shows.
 */
export function useWorkspaceFetch(workspaceSlug: string | undefined): WorkspaceAccess {
  const {
    workspaces,
    getWorkspaceBySlug,
    preferences: { fetchPreferences },
  } = useWorkspace();
  const { fetchProjects } = useProject();
  const { fetchWorkspaceStates } = useProjectState();
  const listed = useWorkspacesFetch();
  // the address's workspace is the caller's once his list has it
  const workspace = workspaceSlug === undefined ? null : getWorkspaceBySlug(workspaceSlug);
  useWorkspaceMembersFetch(workspace);
  useSessionSWR(workspace && ["WORKSPACE_PREFERENCES", workspace.id, workspace.slug], (id, slug) =>
    fetchPreferences({ id, slug })
  );
  useSessionSWR(workspace && ["PROJECTS", workspace.id, workspace.slug], (id, slug) => fetchProjects({ id, slug }));
  useSessionSWR(workspace && ["WORKSPACE_STATES", workspace.id, workspace.slug], (id, slug) =>
    fetchWorkspaceStates({ id, slug })
  );
  if (listed.error) return { kind: "unavailable", retry: () => void listed.mutate() };
  if (workspaces === undefined) return { kind: "loading" };
  if (workspace === null) return { kind: "not-found", hasWorkspaces: workspaces.length > 0 };
  return { kind: "ready", workspace };
}
