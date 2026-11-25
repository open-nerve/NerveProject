/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useLayoutEffect, useMemo, useRef, useState } from "react";
import { isEqual } from "lodash-es";
// nerve imports
import type { View, ViewFilters, ViewUpdate } from "@nerve/api-client";
import type { TSaveViewOptions, TUpdateViewOptions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IIssueFilters, TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { deepCompareFilterExpressions } from "@nerve/utils";
// components
import type { ViewContent } from "@/components/views/view-content";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
import { toFilterExpression, toProjectFilters } from "@/lib/issue-filters";
import { viewsCreatableBy } from "@/lib/views-entry";
// store
import type { IPagePreferencesFilter } from "@/store/issue/helpers/page-preferences-filter.store";

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

/** Whether the draft's display, in nerve's words, is not the view's. */
const displayDiffers = (draft: Pick<View, "display_filters" | "display_properties">, view: View) =>
  !isEqual(draft.display_filters, view.display_filters) || !isEqual(draft.display_properties, view.display_properties);

/**
 * Save view and Update view of a project's list page (P6a spec 2.7, R7–R9, R21, R30). Save view, to whom nerve lets
 * make a view (viewsCreatableBy), opens the view's dialog, its name empty, with what the new view keeps: the page's
 * conditions, its own term, and the display its preferences confirmed, in nerve's words (R8). Update view, to the
 * view's owner, an admin or a member (nerve's view.update; no lock), while the page's draft holds a change of the view's
 * filters or display (the display compared in nerve's words, as the page shows it: confirmed is not observable), sends
 * the page's filters and its draft's display as the change's turn comes, and waits for nerve: confirmed, it says so,
 * the draft and the page's display become the answer, and the filters too where nerve gave other conditions (their
 * baseline then the answer, set by the filter as this settles); refused, it says why by nerve's code and rejects, the
 * draft and the baseline as they were. Its follow-ups are the page's while it is mounted, and its session's
 * (followInSession, which never rejects: the refusal it records is thrown after it).
 */
export function useProjectViewSaving(page: Page) {
  const { workspaceSlug, projectId, entityId, shown, enableSaveView, enableUpdateView, preferences, viewTerms } = page;
  const { t } = useTranslation();
  // the latest refusal toast, which a late answer's follow-up says with (a new one each render)
  const refusal = useRefusalToast();
  const toastRefusal = useRef(refusal);
  toastRefusal.current = refusal;
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { data: currentUser } = useUser();
  const views = useViews();
  const { issuesFilter: drafts } = useIssues(EIssuesStoreType.PROJECT_VIEW);
  const { getFilter } = useWorkItemFilters();
  const [saveAs, setSaveAs] = useState<ViewContent | null>(null);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  // derived values: the route's project and the caller's role in it, never the router store's
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const view = enableUpdateView ? views.getViewById(entityId) : undefined;
  const canSave = !!enableSaveView && viewsCreatableBy(project, role);
  const canUpdate =
    view !== undefined &&
    currentUser !== undefined &&
    view.owned_by_id === currentUser.id &&
    viewsCreatableBy(project, role);
  const { displayFilters, displayProperties } = shown ?? {};
  const hasAdditionalChanges = useMemo(() => {
    const draft = displayFilters && displayProperties ? drafts.confirmed(entityId) : undefined;
    return draft !== undefined && view !== undefined && displayDiffers(draft, view);
  }, [drafts, entityId, view, displayFilters, displayProperties]);

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

  const onViewUpdate = useCallback(
    async (expression: TWorkItemFilterExpression) => {
      // the body as the change's turn comes: the draft's display then (spec 2.7)
      const body = (): ViewUpdate => {
        const draft = drafts.confirmed(entityId);
        return {
          filters: toProjectFilters(expression),
          display_filters: draft?.display_filters,
          display_properties: draft?.display_properties,
        };
      };
      let refused: { error: unknown } | undefined;
      await followInSession(() => views.update(entityId, body), {
        done: async (answer) => {
          if (!mounted.current) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("project_views.action.update.success"),
          });
          drafts.adopt(entityId, answer);
          await drafts.fetchFilters(workspaceSlug, entityId);
          // the conditions nerve gave, where they are not the page's (equal ones keep their controls and the focus)
          const filter = getFilter(EIssuesStoreType.PROJECT_VIEW, entityId);
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
    },
    [drafts, entityId, views, t, workspaceSlug, getFilter]
  );

  const saveViewOptions = useMemo(
    (): TSaveViewOptions<TWorkItemFilterExpression> => ({ isDisabled: !canSave, onViewSave }),
    [canSave, onViewSave]
  );
  const updateViewOptions = useMemo(
    (): TUpdateViewOptions<TWorkItemFilterExpression> => ({
      isDisabled: !canUpdate,
      hasAdditionalChanges,
      onViewUpdate,
    }),
    [canUpdate, hasAdditionalChanges, onViewUpdate]
  );

  return { saveViewOptions, updateViewOptions, saveAs, closeSaveAs: () => setSaveAs(null) };
}
