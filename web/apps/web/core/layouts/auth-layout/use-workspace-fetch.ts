/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/** Fetched once a session: a page that mounts again shows what the stores have. */
const ONCE = { revalidateIfStale: false, revalidateOnFocus: false };

/**
 * The workspace side of what a page of a workspace fetches as it mounts (M3 design 3.1, 7.1): the caller's
 * workspaces, which decide whether he may see the address's one; once his list has it, its members and his
 * navigation settings in it. Gives the list's response, whose failure the page shows.
 */
export function useWorkspaceFetch(workspaceSlug: string | undefined): SWRResponse<Workspace[] | undefined> {
  const {
    fetchWorkspaces,
    getWorkspaceBySlug,
    preferences: { fetchPreferences },
  } = useWorkspace();
  const {
    workspace: { fetchWorkspaceMembers },
  } = useMember();
  // the address's workspace is the caller's once his list has it
  const isMember = workspaceSlug !== undefined && getWorkspaceBySlug(workspaceSlug) !== null;
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces(), { ...ONCE, shouldRetryOnError: false });
  useSessionSWR(isMember ? ["WORKSPACE_MEMBERS", workspaceSlug] : null, (slug) => fetchWorkspaceMembers(slug), ONCE);
  useSessionSWR(isMember ? ["WORKSPACE_PREFERENCES", workspaceSlug] : null, (slug) => fetchPreferences(slug), ONCE);
  return listed;
}
