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

/**
 * The one fetch of the caller's workspaces (M3 design 7.1), which the landing, a workspace's pages, the onboarding and
 * the profile's settings share: the session's list, fetched as each mounts; nothing while wanted is false.
 */
export function useWorkspacesFetch(wanted = true): SWRResponse<Workspace[] | undefined> {
  const { fetchWorkspaces } = useWorkspace();
  return useSessionSWR(wanted ? ["WORKSPACES"] : null, () => fetchWorkspaces());
}
