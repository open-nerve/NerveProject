/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The projects page's own fetch as it mounts (M3 design 7.1): once the caller's list has the address's workspace, its
 * archived projects, keyed by its id, from its slug's address; none for a workspace not his. Its projects that are not
 * archived are the fetch of every page of the workspace (useWorkspaceFetch); the page shows its projects once both
 * lists are there (filteredProjectIds).
 */
export function useArchivedProjectsFetch(workspaceSlug: string | undefined): void {
  const { getWorkspaceBySlug } = useWorkspace();
  const { fetchArchivedProjects } = useProject();
  const workspace = workspaceSlug === undefined ? null : getWorkspaceBySlug(workspaceSlug);
  useSessionSWR(workspace && ["ARCHIVED_PROJECTS", workspace.id, workspace.slug], (id, slug) =>
    fetchArchivedProjects({ id, slug })
  );
}
