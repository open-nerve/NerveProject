/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect } from "react";
// nerve imports
import type { DefaultWorkspaceViewKey } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
// store
import { workspacePageId } from "@/store/issue/workspace/page-id";

/**
 * A default view's page's read of its preferences (M7/P6b spec 2.4, 2.8; re-97 I2): the display the browser keeps and
 * this entry's filters, none yet, read as the page mounts, each mount its own, and ended as it unmounts (its filters
 * last for one entry). Whether they are held: the page draws its list once they are. The read is the store's, never
 * SWR's: within SWR's dedupe window a mount again of the same key would call no read, and the page, whose preferences
 * its unmount ended, would wait for nothing. The preferences are the browser's and memory's, read at once; the read
 * fails only when the session changed meanwhile, whose pages are mounted afresh: no failure of this one.
 */
export function useDefaultViewRead(workspaceSlug: string, key: DefaultWorkspaceViewKey): boolean {
  const { issuesFilter } = useIssues(EIssuesStoreType.GLOBAL);
  const page = workspacePageId(workspaceSlug, key);
  useEffect(() => {
    void issuesFilter.fetchFilters(workspaceSlug, page).catch(sessionChangedIsNoFailure);
    return () => issuesFilter.discard(page);
  }, [issuesFilter, workspaceSlug, page]);
  return issuesFilter.getIssueFilters(page) !== undefined;
}

/** The end of a read the session changed under: its pages are the new session's. Any other failure is the read's. */
function sessionChangedIsNoFailure(error: unknown): void {
  if (!(error instanceof SessionChangedError)) throw error;
}
