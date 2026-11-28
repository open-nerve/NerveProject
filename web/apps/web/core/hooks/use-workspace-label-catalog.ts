/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * Reads the workspace's label catalog into the label store as a page mounts (M7/P6b spec 2.2, R17): the labels of
 * its projects the caller is a member of, unarchived, which the store gives each such project. The workspace is the
 * route's, by its slug, once the caller's list has it; enabled is the page's own gate. Its readers are the pages that
 * list work items across the projects, each behind its gate: the workspace's view page, the profile's work items and
 * the drafts page; the workspace's shell does not read it.
 */
export function useWorkspaceLabelCatalog(workspaceSlug: string, enabled: boolean): void {
  const { getWorkspaceBySlug } = useWorkspace();
  const { fetchWorkspaceLabels } = useLabel();
  const workspace = getWorkspaceBySlug(workspaceSlug);
  useSessionSWR(enabled && workspace ? ["WORKSPACE_LABELS", workspace.id, workspace.slug] : null, (id, slug) =>
    fetchWorkspaceLabels({ id, slug })
  );
}
