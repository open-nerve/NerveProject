/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef } from "react";
import { cloneDeep } from "lodash-es";
import { runInAction } from "mobx";
import type { EIssuesStoreType, TWorkItemFilterExpression } from "@nerve/types";
import { deepCompareFilterExpressions } from "@nerve/utils";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import type { IPagePreferencesFilter } from "@/store/issue/helpers/page-preferences-filter.store";

/** The page's follow of a change of its preferences (usePagePreferencesFollow). */
type Follow = (slug: string, id: string, change: () => Promise<void>) => Promise<void>;

/**
 * The save of a list page's filter conditions, whose preferences hold them (the project's page, a cycle's or a
 * module's: P5b spec 2.4, P6b spec 2.4), made under the page's follow. Once nerve confirms them and the page's draft is
 * still the one sent, the confirmation is the draft's saved baseline: as it is when nerve accepted the conditions
 * unchanged, else as nerve gave them.
 */
export function usePageFilterExpression(
  storeType: EIssuesStoreType.PROJECT | EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE,
  workspaceSlug: string | undefined,
  id: string | undefined,
  issuesFilter: IPagePreferencesFilter,
  follow: Follow
) {
  const { getFilter } = useWorkItemFilters();
  const applyingConfirmation = useRef(false);
  const editSequence = useRef(0);
  return useCallback(
    (expression: TWorkItemFilterExpression) => {
      if (applyingConfirmation.current || !workspaceSlug || !id) return;
      const sequence = ++editSequence.current;
      const filter = getFilter(storeType, id);
      const submitted = JSON.stringify(filter?.expression);
      void follow(workspaceSlug, id, async () => {
        await issuesFilter.updateFilterExpression(workspaceSlug, id, expression);
        // Capture the entire draft, including unfinished inputs that have not notified a save yet.
        if (
          sequence === editSequence.current &&
          filter &&
          getFilter(storeType, id) === filter &&
          JSON.stringify(filter.expression) === submitted
        ) {
          applyingConfirmation.current = true;
          try {
            const confirmed = issuesFilter.getIssueFilters(id)?.richFilters ?? {};
            if (deepCompareFilterExpressions(filter.expression, filter.adapter.toInternal(confirmed))) {
              // The server accepted these conditions unchanged. Keep their DOM identity/focus while advancing
              // the saved baseline; rebuilding equivalent conditions would discard the active keyboard control.
              runInAction(() => {
                filter.initialFilterExpression = cloneDeep(filter.expression);
              });
            } else {
              filter.resetExpression(confirmed);
            }
          } finally {
            applyingConfirmation.current = false;
          }
        }
      });
    },
    [storeType, workspaceSlug, id, follow, issuesFilter, getFilter]
  );
}
