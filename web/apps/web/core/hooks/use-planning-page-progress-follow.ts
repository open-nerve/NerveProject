/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { useParams } from "react-router";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useIssues } from "@/hooks/store/use-issues";
import { followInSession } from "@/lib/in-session";
import { PLANNING_PAGE_DOMAINS, type PlanningPageStoreType } from "./planning-page-domains";

/**
 * Sends a write of a cycle's or a module's page's list, and reads the route cycle's or module's progress again once
 * nerve confirms it (P5b spec 2.6, R10; re-66 L5; P6b spec 2.2, R13): a creation, a change, a removal from the page, a
 * deletion, an archiving or a drop may change what the progress counts, which the page never works out itself; the
 * store keeps the newest answer. A write whose answer may not be the page's says by concerns which is (a form's
 * creation in another cycle is not, P7a spec 2.6). It follows the write only in the session and page it was sent from: the route's cycle
 * or module of its project, held from the sending to the answer (another one and back is not the same page), so a late
 * answer reads nothing for the page then shown. The write's own answer or refusal goes to its caller untouched; a
 * failed read leaves the progress as the page shows it, until the page reads it again.
 */
export function usePlanningPageProgressFollow(storeType: PlanningPageStoreType) {
  const domain = PLANNING_PAGE_DOMAINS[storeType];
  const params = useParams();
  const { workspaceSlug, projectId } = params;
  const entityId = params[domain.param];
  const scope = useDetailScope(workspaceSlug ?? "", projectId, JSON.stringify([`${domain.name}-progress`, entityId]));
  const { issues } = useIssues(storeType);
  return useCallback(
    <T>(write: () => Promise<T>, concerns: (answer: T) => boolean = () => true): Promise<T> => {
      if (!entityId) return write();
      const owned = scope();
      const progress = domain.progress(issues.rootIssueStore.rootStore);
      const sent = write();
      void followInSession(() => sent, {
        done: (answer) => {
          if (owned() && concerns(answer)) void progress.fetchProgress(entityId).catch(() => undefined);
        },
        failed: () => undefined,
      });
      return sent;
    },
    [scope, issues, domain, entityId]
  );
}
