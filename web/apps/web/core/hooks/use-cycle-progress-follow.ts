/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useIssues } from "@/hooks/store/use-issues";
import { followInSession } from "@/lib/in-session";

/**
 * Sends a write of the cycle page's list, and reads the route cycle's progress again once nerve confirms it (P5b spec
 * 2.6, R10; re-66 L5): a creation, a change, a removal from the cycle, a deletion, an archiving or a drop may change
 * what the progress counts, which the page never works out itself; the store keeps the newest answer. It follows the
 * write only in the session and page it was sent from: the route's cycle of its project, held from the sending to the
 * answer (another cycle and back is not the same page), so a late answer reads nothing for the page then shown. The
 * write's own answer or refusal goes to its caller untouched; a failed read leaves the progress as the page shows it,
 * until the page reads it again.
 */
export function useCycleProgressFollow() {
  const { workspaceSlug, projectId, cycleId } = useParams();
  const scope = useDetailScope(workspaceSlug ?? "", projectId, JSON.stringify(["cycle-progress", cycleId]));
  const { issues } = useIssues(EIssuesStoreType.CYCLE);
  return useCallback(
    <T>(write: () => Promise<T>): Promise<T> => {
      if (!cycleId) return write();
      const owned = scope();
      const { cycles } = issues.rootIssueStore.rootStore.planning;
      const sent = write();
      void followInSession(() => sent, {
        done: () => {
          if (owned()) void cycles.fetchProgress(cycleId).catch(() => undefined);
        },
        failed: () => undefined,
      });
      return sent;
    },
    [scope, issues, cycleId]
  );
}
