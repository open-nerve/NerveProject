/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useMemo, useState } from "react";
// nerve imports
import type { TSaveViewOptions } from "@nerve/constants";
import type { IIssueFilters, TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { toWorkspaceFilters } from "@/lib/issue-filters";
import { workspaceViewsCreatableBy } from "@/lib/views-entry";
// store
import { defaultViewOfPage, ownTerm } from "@/store/issue/workspace/page-id";
import type { ViewContent } from "@/store/views/view-content";
// local imports
import { useViewUpdate } from "./use-view-update";

/** What a workspace's page gives its filters of Save view and Update view (M7/P6b spec 2.9). */
export type TWorkspaceViewSavingProps = {
  /** Offers Save view: a new view of the workspace's own, of what the page shows. */
  enableSaveView?: boolean;
  /** Offers Update view: the page is a workspace's own view's (its id the filters' entity), whose draft it changes. */
  enableUpdateView?: boolean;
};

type Page = TWorkspaceViewSavingProps & {
  /** The route's workspace. */
  workspaceSlug: string;
  /** The filters' entity, the page's id: a default view's in its workspace, a saved view's id (page-id.ts). */
  entityId: string;
  /** The page's filters as it shows them (filters[id]: observable, as its preferences' confirmed copy is not). */
  shown: IIssueFilters | undefined;
};

/**
 * Save view and Update view of a workspace's view's page (M7/P6b spec 2.7, 2.9, R8, R10), to whom nerve lets write the
 * workspace's own views, its admins and members (workspaceViewsCreatableBy, by the route's workspace), never its guest.
 * Save view, on each of the five pages, opens the view's dialog, its name empty, with what the new view keeps: the
 * page's conditions, a default view's own term besides them (its `me`, which the page's query fixes), and the display
 * the page holds, in nerve's words, as the spreadsheet, the one layout the page draws, whatever layout a view held.
 * Update view, of a saved view's page, is the views' pages' own (useViewUpdate).
 */
export function useWorkspaceViewSaving(page: Page) {
  const { workspaceSlug, entityId, shown, enableSaveView, enableUpdateView } = page;
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { issuesFilter: preferences } = useIssues(EIssuesStoreType.GLOBAL);
  const [saveAs, setSaveAs] = useState<ViewContent | null>(null);
  // derived values: the caller's role in the route's workspace, never the router store's
  const writes = workspaceViewsCreatableBy(getWorkspaceRoleByWorkspaceSlug(workspaceSlug));
  const canSave = !!enableSaveView && writes;
  const updateViewOptions = useViewUpdate({
    storeType: EIssuesStoreType.GLOBAL,
    workspaceSlug,
    viewId: entityId,
    enabled: !!enableUpdateView,
    writes,
    shown,
  });

  const onViewSave = useCallback(
    (expression: TWorkItemFilterExpression) => {
      const held = preferences.confirmed(entityId);
      const key = defaultViewOfPage(entityId);
      setSaveAs({
        filters: { ...toWorkspaceFilters(expression), ...(key ? ownTerm(key) : {}) },
        display_filters: held && { ...held.display_filters, layout: "spreadsheet" },
        display_properties: held?.display_properties,
      });
    },
    [preferences, entityId]
  );

  const saveViewOptions = useMemo(
    (): TSaveViewOptions<TWorkItemFilterExpression> => ({ isDisabled: !canSave, onViewSave }),
    [canSave, onViewSave]
  );

  return { saveViewOptions, updateViewOptions, saveAs, closeSaveAs: () => setSaveAs(null) };
}
