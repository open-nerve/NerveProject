/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
import type { Workspace, WorkspaceMember } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The one fetch of a workspace's members (M3 design 7.1), which a workspace's pages, its members settings and a
 * profile share: given a workspace of the caller's list, its members, keyed by its id, from its slug's address; given
 * none (the address names no workspace of his, or not yet), nothing, as a non-member may not read them.
 */
export function useWorkspaceMembersFetch(
  workspace: Workspace | null
): SWRResponse<Record<string, WorkspaceMember> | undefined> {
  const {
    workspace: { fetchWorkspaceMembers },
  } = useMember();
  return useSessionSWR(workspace && ["WORKSPACE_MEMBERS", workspace.id, workspace.slug], (id, slug) =>
    fetchWorkspaceMembers({ id, slug })
  );
}
