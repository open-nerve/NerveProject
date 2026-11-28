/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useMemo, useState } from "react";
// nerve imports
import type { ViewFilters } from "@nerve/api-client";
import type { TSaveViewOptions } from "@nerve/constants";
import type { IIssueFilters, TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
// components
import type { ViewContent } from "@/components/views/view-content";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { toProjectFilters } from "@/lib/issue-filters";
import { viewsCreatableBy } from "@/lib/views-entry";
// store
import type { IPagePreferencesFilter } from "@/store/issue/helpers/page-preferences-filter.store";
// local imports
import { useViewUpdate } from "./use-view-update";

/** What a list page gives its filters of Save view and Update view (P6a spec 2.7, R7–R9). */
export type TProjectViewSavingProps = {
  /** Offers Save view: a new view of what the page shows. */
  enableSaveView?: boolean;
  /** Offers Update view: the page is a view's (its id the filters' entity), whose draft the page changes. */
  enableUpdateView?: boolean;
  /** The page's own preferences, whose confirmed display a new view keeps (R8). */
  preferences?: IPagePreferencesFilter;
  /** What a new view keeps besides the page's filters: a cycle's or a module's page's own term (R7). */
  viewTerms?: (id: string) => ViewFilters;
};

type Page = TProjectViewSavingProps & {
  /** The page's address: the route's workspace and project, and the filters' entity, the page's id. */
  workspaceSlug: string;
  projectId: string;
  entityId: string;
  /** The page's filters as it shows them (filters[id]: observable, as its preferences' confirmed copy is not). */
  shown: IIssueFilters | undefined;
};

/**
 * Save view and Update view of a project's list page (P6a spec 2.7, R7–R9, R21, R30; M7/P6b spec 2.9). Save view, to
 * whom nerve lets make a view (viewsCreatableBy), opens the view's dialog, its name empty, with what the new view
 * keeps: the page's conditions, its own term, and the display its preferences confirmed, in nerve's words (R8). Update
 * view, of a project's view's page, is the views' pages' own (useViewUpdate), to the view's owner while he makes the
 * project's views.
 */
export function useProjectViewSaving(page: Page) {
  const { workspaceSlug, projectId, entityId, shown, enableSaveView, enableUpdateView, preferences, viewTerms } = page;
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const [saveAs, setSaveAs] = useState<ViewContent | null>(null);
  // derived values: the route's project and the caller's role in it, never the router store's
  const writes = viewsCreatableBy(
    getProjectById(projectId),
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
  );
  const canSave = !!enableSaveView && writes;
  const updateViewOptions = useViewUpdate({
    storeType: EIssuesStoreType.PROJECT_VIEW,
    workspaceSlug,
    viewId: entityId,
    enabled: !!enableUpdateView,
    writes,
    shown,
  });

  const onViewSave = useCallback(
    (expression: TWorkItemFilterExpression) => {
      const confirmed = preferences?.confirmed(entityId);
      setSaveAs({
        filters: { ...toProjectFilters(expression), ...viewTerms?.(entityId) },
        display_filters: confirmed?.display_filters,
        display_properties: confirmed?.display_properties,
      });
    },
    [preferences, viewTerms, entityId]
  );

  const saveViewOptions = useMemo(
    (): TSaveViewOptions<TWorkItemFilterExpression> => ({ isDisabled: !canSave, onViewSave }),
    [canSave, onViewSave]
  );

  return { saveViewOptions, updateViewOptions, saveAs, closeSaveAs: () => setSaveAs(null) };
}
