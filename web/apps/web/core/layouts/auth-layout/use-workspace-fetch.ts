/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/** Fetched once a session: a page that mounts again shows what the stores have. */
const ONCE = { revalidateIfStale: false, revalidateOnFocus: false };

/**
 * The workspace side of what a page of a workspace fetches as it mounts (M3 design 3.1, 7.1): the caller's
 * workspaces, which decide whether he may see the address's one. Gives the list's response, whose failure the page
 * shows.
 */
export function useWorkspaceFetch(): SWRResponse<Workspace[] | undefined> {
  const { fetchWorkspaces } = useWorkspace();
  return useSessionSWR(["WORKSPACES"], () => fetchWorkspaces(), { ...ONCE, shouldRetryOnError: false });
}
