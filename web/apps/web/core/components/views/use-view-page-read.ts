/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useState } from "react";
// nerve imports
import type { View } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { problemCode } from "@/lib/api-error";
import { useSessionSWR } from "@/lib/use-session-swr";
import { viewEntryOn } from "@/lib/views-entry";
// store
import { NoViewDraftError } from "@/store/issue/helpers/view-draft.service";

/**
 * Where a view's page is with its own read of the view: reading it; nerve finds no such view; why nerve could not read
 * it, with a retry; read: the page's draft seeded, or none for a view deleted meanwhile, which ViewVisit shows as
 * not found unless its own page sent the deletion and is leaving.
 */
export type ViewPageRead =
  | { state: "reading" }
  | { state: "not-found" }
  | { state: "failed"; error: unknown; retry: () => void }
  | { state: "read" };

/**
 * A view's page's read of its view (P6a spec 2.6, R2; re-96 I1, I2), which seeds the page's draft of it: each mount of
 * the page makes its own as it mounts (mutate, which never joins an earlier read), and only an explicit retry reads
 * again, never SWR on a mount or a reconnect. An answer SWR keeps of an earlier mount is not this mount's: within SWR's
 * dedupe window it would be taken for one, and the page, whose draft the earlier mount ended, would wait for nothing
 * (use-session-swr.ts). It ends the page's draft as the page unmounts (a leave, another view, another project), any
 * read of it still out with it: entering again shows nothing of the old draft until nerve answers anew. The draft
 * service's refusal (NoViewDraftError) is no failure of the read (noDraftIsNoFailure).
 */
export function useViewPageRead(workspaceSlug: string, viewId: string): ViewPageRead {
  const { issuesFilter } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  const read = useSessionSWR(
    ["PROJECT_VIEW_PAGE", workspaceSlug, viewId],
    (slug, id) => issuesFilter.fetchFilters(slug, id).catch(noDraftIsNoFailure),
    { revalidateOnMount: false, revalidateOnReconnect: false }
  );
  const { mutate } = read;
  // whether this mount's own read has answered: what SWR keeps of an earlier mount's is not its own
  const [answered, setAnswered] = useState(false);
  useEffect(() => {
    let mounted = true;
    // mutate reads the view anew, never joining an earlier read, and settles once SWR holds its answer
    void mutate().finally(() => {
      if (mounted) setAnswered(true);
    });
    return () => {
      mounted = false;
      issuesFilter.discard(viewId);
    };
  }, [mutate, issuesFilter, viewId]);

  if (!answered || read.isValidating) return { state: "reading" };
  const { error } = read;
  if (error === undefined) return { state: "read" };
  if (problemCode(error) === "navigation.view_not_found") return { state: "not-found" };
  return { state: "failed", error, retry: () => void mutate() };
}

/**
 * The end of a read of the view that made no draft (NoViewDraftError, view-draft.service.ts): its page left before
 * nerve answered, or the view was deleted meanwhile, which ViewVisit shows as not found unless its own page sent the
 * deletion. Either is no failure of the read. As SWR's error it would reach not the page that asked but the mount that
 * holds the read's key then: the same view entered again, whose own read, sent later, nerve may answer first (review
 * m1). Ended so, a read SWR finds overtaken by a later one writes nothing. Any other failure is the read's.
 */
function noDraftIsNoFailure(error: unknown): void {
  if (!(error instanceof NoViewDraftError)) throw error;
}

/**
 * The view a view's page shows, of the route's project (P6a spec 2.6): once the page's read of it has seeded the page's
 * draft (the page's filters are then the draft's), while the ways into the project's views are the caller's
 * (viewEntryOn); none while the page reads it, finds none, could not read it, or the project's views are off. The page's
 * header and its small screen's header show the view's controls while there is one, and else only the crumbs to the
 * project's views.
 */
export function useShownView(
  workspaceSlug: string,
  projectId: string,
  viewId: string
): { entry: boolean; view: View | undefined } {
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getViewById } = useViews();
  const { issuesFilter } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  const entry = viewEntryOn(
    getProjectById(projectId),
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
  );
  const view = getViewById(viewId);
  const shown =
    entry && view?.project_id === projectId && issuesFilter.getIssueFilters(viewId) !== undefined ? view : undefined;
  return { entry, view: shown };
}
