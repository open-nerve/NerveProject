/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// store
import type { TPeekIssue } from "@/store/issue/issue-details/root.store";
// local imports
import type { TIssueOperations } from "../issue-detail/root";

/**
 * The peek's read of a work item into the stores (M4; P5b spec 2.5, R16, R27): the session's fetch of the work item
 * of the peek's workspace and project, by the caller's permissions there (their identity), which fetch (the issue
 * operations') makes. The peek reads with the session's defaults: as it mounts, and again as the browser reconnects.
 * The notifications page's preview, which judges by this read whether to peek at the work item or embed its intake
 * detail, reads with the same key, so that a peek mounting right after joins that read: its mount read and its
 * reconnect read off (config), made with mutate, each mount's own.
 */
export function usePeekIssueRead(
  peek: Pick<TPeekIssue, "workspaceSlug" | "projectId" | "issueId">,
  fetch: TIssueOperations["fetch"],
  config?: Parameters<typeof useSessionSWR>[2]
): SWRResponse<void> {
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(peek.workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [peek.projectId]);
  return useSessionSWR(
    ["peek-issue", identity, peek.workspaceSlug, peek.projectId, peek.issueId],
    (_identity, ws, project, id) => fetch(ws, project, id),
    config
  );
}
