/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef } from "react";
import { useNavigate } from "react-router";
import type { TWorkItemFilterCondition } from "@nerve/shared-state";
import { EIssuesStoreType } from "@nerve/types";
// components
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { peekOnEntry } from "../peek-on-entry";

/**
 * Takes the cycle's page to the work items of one condition, and there previews the work item issueId, if given (P5b
 * spec 2.7, 2.8, R8).
 */
export type EnterFiltered = (condition: TWorkItemFilterCondition, issueId?: string) => void;

/**
 * How a row of the card of the cycle in progress takes its cycle's page to the row's work items (P5b spec 2.8, R8):
 * the page's filters are replaced by the one condition, its conditions alone, never merged with those it had. Off the cycle's page there is no instance of its filters, so the change replaces the
 * preferences' whole field, which needs nothing read (one PATCH, P5b spec 2.4, re-66 M1). Once nerve confirms it the
 * page is entered, with the work item to preview there (peekOnEntry); a refusal is said, and the list page stays. One
 * change at a time: a click while one is out does nothing. The follow-ups are those of the session the change was sent
 * in (followInSession) and of the card's page while it shows the cycle (useDetailScope: the card, the caller's
 * permission): late, nerve's answer reaches the store alone.
 */
export function useEnterFiltered(workspaceSlug: string, projectId: string, cycleId: string): EnterFiltered {
  const navigate = useNavigate();
  const { issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  const { updateFilterExpressionFromConditions } = useWorkItemFilters();
  const refusal = useRefusalToast();
  const scope = useDetailScope(workspaceSlug, projectId, JSON.stringify(["cycle-row-filter", cycleId]));
  const out = useRef(false);
  return useCallback(
    (condition, issueId) => {
      if (out.current) return;
      out.current = true;
      const current = scope();
      void followInSession(
        () =>
          updateFilterExpressionFromConditions(EIssuesStoreType.CYCLE, cycleId, [condition], (expression) =>
            issuesFilter.updateFilterExpression(workspaceSlug, cycleId, expression)
          ),
        {
          done: () => {
            if (!current()) return;
            const page = `/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`;
            return navigate(page, issueId ? { state: peekOnEntry(issueId) } : undefined);
          },
          failed: (error) => {
            if (current()) refusal(error);
          },
        }
      ).finally(() => {
        out.current = false;
      });
    },
    [workspaceSlug, projectId, cycleId, navigate, issuesFilter, updateFilterExpressionFromConditions, refusal, scope]
  );
}
