/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
// components
import { openIssuePeek } from "@/components/issues/peek-overview/selection";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

/** The navigation's state by which a page asks a cycle's page to preview a work item of it as it enters. */
type PeekOnEntryState = { peekIssue: string };

/** The state of a navigation to a cycle's page that previews the work item issueId there (PeekOnEntry). */
export const peekOnEntry = (issueId: string): PeekOnEntryState => ({ peekIssue: issueId });

/** The work item a navigation's state asks the page to preview, if any. */
const askedOf = (state: unknown): string | undefined =>
  typeof state === "object" && state !== null && "peekIssue" in state && typeof state.peekIssue === "string"
    ? state.peekIssue
    : undefined;

type Props = { workspaceSlug: string; projectId: string };

/**
 * Previews the work item that the navigation into a cycle's page asked for (peekOnEntry: the card of the cycle in
 * progress, by its urgent and high row; P5b spec 2.7, re-66 M1). The preview is the issue detail's selection, which the
 * host mounted then takes (peek-overview/root.tsx), so it is asked for here, beside the page's own host, once both are
 * in place, and not before the page is there. The ask is then taken off the history's entry: a reload, a Back and a
 * Forward do not preview it again.
 */
export function PeekOnEntry({ workspaceSlug, projectId }: Props) {
  const location = useLocation();
  const navigate = useNavigate();
  const detail = useIssueDetail();
  const issueId = askedOf(location.state);
  useEffect(() => {
    if (!issueId) return;
    openIssuePeek(detail, { workspaceSlug, projectId, issueId });
    void navigate(
      { pathname: location.pathname, search: location.search, hash: location.hash },
      { replace: true, state: null }
    );
  }, [issueId, detail, workspaceSlug, projectId, navigate, location.pathname, location.search, location.hash]);
  return null;
}
