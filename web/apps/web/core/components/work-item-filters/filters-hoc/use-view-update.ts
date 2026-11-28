/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { isEqual } from "lodash-es";
// nerve imports
import type { View, ViewUpdate } from "@nerve/api-client";
import type { TUpdateViewOptions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IIssueFilters, TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { deepCompareFilterExpressions } from "@nerve/utils";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useViews } from "@/hooks/store/use-views";
import { useUser } from "@/hooks/store/user";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
import { toFilterExpression } from "@/lib/issue-filters";

/** The pages of a view, whose draft Update view saves: a project's view's, a workspace's own view's. */
type ViewPageStore = EIssuesStoreType.PROJECT_VIEW | EIssuesStoreType.GLOBAL;

/** What each page says once nerve has changed its view (M7/P6b spec 2.9, R29), its keys whole. */
const UPDATED = {
  [EIssuesStoreType.PROJECT_VIEW]: "project_views.action.update.success",
  [EIssuesStoreType.GLOBAL]: "workspace_views.action.update.success",
} as const;

type ViewPage = {
  /** The page's store type: its draft of the view is that store's preferences of the view's id. */
  storeType: ViewPageStore;
  /** The route's workspace. */
  workspaceSlug: string;
  /** The view the page shows: its id, the filters' entity and its draft's. */
  viewId: string;
  /** Offers Update view: the page is a view's. */
  enabled: boolean;
  /** Whether the caller writes the views of the view's container, by his role there (viewsCreatableBy, …). */
  writes: boolean;
  /** The page's filters as it shows them (filters[id]: observable, as its draft's confirmed copy is not). */
  shown: IIssueFilters | undefined;
};

/** Whether the draft's display, in nerve's words, is not the view's. */
const displayDiffers = (draft: Pick<View, "display_filters" | "display_properties">, view: View) =>
  !isEqual(draft.display_filters, view.display_filters) || !isEqual(draft.display_properties, view.display_properties);

/**
 * Update view of a view's page, a project's or a workspace's own (P6a spec 2.7, R9; M7/P6b spec 2.9, R10): offered to
 * the view's owner while he writes its container's views (writes; no lock) and the page's draft holds a change of the
 * view's filters or display (the display compared in nerve's words, as the page shows it). Its change's body is the
 * page's draft, whole, as the change's turn comes (O1): its filters, display and properties, the draft being what the
 * page lists; the expression the filters row gives on the click is not used. It waits for nerve: confirmed, it says so
 * in the container's words, the draft and the page's filters become the answer, and the filters' expression too where
 * nerve gave other conditions (their baseline then the answer, set by the filter as this settles); refused, it says why
 * by nerve's code and rejects, the draft and the baseline as they were. Its follow-ups are the page's while it is
 * mounted, and its session's (followInSession, which never rejects: the refusal it records is thrown after it).
 */
export function useViewUpdate(page: ViewPage): TUpdateViewOptions<TWorkItemFilterExpression> {
  const { storeType, workspaceSlug, viewId, enabled, writes, shown } = page;
  const { t } = useTranslation();
  // the latest refusal toast, which a late answer's follow-up says with (a new one each render)
  const refusal = useRefusalToast();
  const toastRefusal = useRef(refusal);
  toastRefusal.current = refusal;
  const { data: currentUser } = useUser();
  const views = useViews();
  const { issuesFilter: drafts } = useIssues(storeType);
  const { getFilter } = useWorkItemFilters();
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  // derived values
  const view = enabled ? views.getViewById(viewId) : undefined;
  const canUpdate = view !== undefined && currentUser !== undefined && view.owned_by_id === currentUser.id && writes;
  const { displayFilters, displayProperties } = shown ?? {};
  const hasAdditionalChanges = useMemo(() => {
    const draft = displayFilters && displayProperties ? drafts.confirmed(viewId) : undefined;
    return draft !== undefined && view !== undefined && displayDiffers(draft, view);
  }, [drafts, viewId, view, displayFilters, displayProperties]);

  const onViewUpdate = useCallback(async () => {
    // the body as the change's turn comes: the page's draft then, whole (O1)
    const body = (): ViewUpdate => {
      const draft = drafts.confirmed(viewId);
      // It is never missing on this path (re-97 m3): the read that draws the page's filters, which Update view needs,
      // holds the draft, and nothing deletes it (the page's discard leaves it: a turn that comes once the page left
      // sends the draft it left). Were there none, nothing is sent and Update view fails, as a refusal does.
      if (!draft) throw new Error(`No draft of view ${viewId}`);
      return {
        filters: draft.filters,
        display_filters: draft.display_filters,
        display_properties: draft.display_properties,
      };
    };
    let refused: { error: unknown } | undefined;
    await followInSession(() => views.update(viewId, body), {
      done: async (answer) => {
        if (!mounted.current) return;
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t(UPDATED[storeType]) });
        drafts.adopt(viewId, answer);
        await drafts.fetchFilters(workspaceSlug, viewId);
        // the conditions nerve gave, where they are not the page's (equal ones keep their controls and the focus)
        const filter = getFilter(storeType, viewId);
        const answered = toFilterExpression(answer.filters);
        if (filter && !deepCompareFilterExpressions(filter.expression, filter.adapter.toInternal(answered)))
          filter.resetExpression(answered);
      },
      failed: (error) => {
        refused = { error };
        if (mounted.current) toastRefusal.current(error);
      },
    });
    if (refused) throw refused.error;
  }, [drafts, viewId, views, t, storeType, workspaceSlug, getFilter]);

  return useMemo(
    () => ({ isDisabled: !canUpdate, hasAdditionalChanges, onViewUpdate }),
    [canUpdate, hasAdditionalChanges, onViewUpdate]
  );
}
