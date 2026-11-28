/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { TWorkItemFilterExpression } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// components
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useViews } from "@/hooks/store/use-views";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { useGlobalPreferencesFollow } from "@/hooks/use-global-preferences-follow";
import { useWorkspaceLabelCatalog } from "@/hooks/use-workspace-label-catalog";
// store
import { defaultViewOfPage, workspacePageId, workspacePageWords } from "@/store/issue/workspace/page-id";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { DraftBusyContext } from "../draft-busy";
import { WorkspaceSpreadsheetRoot } from "../spreadsheet/roots/workspace-root";

type Props = {
  /** The page's address, as the route gives it: the workspace, and its view (a default view's key or a view's id). */
  workspaceSlug: string;
  globalViewId: string;
};

/**
 * A workspace's view's page's work items (M7/P6b spec 2.3, 2.4, 2.8, R1, R7, R8), of the route's workspace and view:
 * the workspace's list under the page's preferences (page-id.ts), by GLOBAL, once the page has read them. Then the
 * page's filters, by the words the page filters by (a default view's own left out), the row of their conditions, the
 * spreadsheet whatever layout a view holds, and the page's own preview, whose host is the page's. A change of the
 * filters is the page's preferences' alone, followed by a read of the list: nothing is sent. Save view makes a view of
 * the workspace's own of what the page shows, on each page; Update view saves a saved view's page's draft (spec 2.9).
 * While it is out, the row and the table's ways that change the draft stop (DraftBusyContext, as P6a R29); a default
 * view's page has no view, so nothing of it is ever being saved. It reads the workspace's label catalog, whose labels
 * the filters, the rows and their groups name, for any role of the workspace (spec 2.2).
 */
export const WorkspaceViewLayoutRoot = observer(function WorkspaceViewLayoutRoot(props: Props) {
  const { workspaceSlug, globalViewId } = props;
  // store hooks
  const { issues, issuesFilter } = useIssues(EIssuesStoreType.GLOBAL);
  const { isUpdating } = useViews();
  // hooks
  const follow = useGlobalPreferencesFollow();
  useWorkspaceLabelCatalog(workspaceSlug, true);
  // derived values
  const page = workspacePageId(workspaceSlug, globalViewId);
  const updateFilters = useCallback(
    (expression: TWorkItemFilterExpression) =>
      void follow(workspaceSlug, page, () => issuesFilter.updateFilterExpression(workspaceSlug, page, expression)),
    [follow, issuesFilter, workspaceSlug, page]
  );
  const workItemFilters = issuesFilter.getIssueFilters(page);
  const busy = isUpdating(page);

  if (!workItemFilters) return null;

  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.GLOBAL}>
      <WorkspaceLevelWorkItemFiltersHOC
        entityType={EIssuesStoreType.GLOBAL}
        entityId={page}
        filtersToShowByLayout={workspacePageWords(globalViewId)}
        initialWorkItemFilters={workItemFilters}
        updateFilters={updateFilters}
        workspaceSlug={workspaceSlug}
        enableSaveView
        enableUpdateView={defaultViewOfPage(page) === undefined}
      >
        {({ filter: pageFilter }) => (
          <DraftBusyContext.Provider value={busy}>
            <div className="relative flex h-full w-full flex-col overflow-hidden">
              {pageFilter && <WorkItemFiltersRow filter={pageFilter} disabled={busy} />}
              <div className="relative h-full w-full overflow-auto bg-surface-1">
                {/* mutation loader */}
                {issues.getIssueLoader() === "mutation" && (
                  <div className="shadow-sm fixed top-[70px] right-[20px] z-20 flex h-[40px] w-[40px] items-center justify-center rounded-sm bg-layer-1">
                    <Spinner className="h-4 w-4" />
                  </div>
                )}
                <WorkspaceSpreadsheetRoot workspaceSlug={workspaceSlug} />
              </div>
              <IssuePeekOverview hostScope={JSON.stringify(["workspace-view", workspaceSlug, globalViewId])} />
            </div>
          </DraftBusyContext.Provider>
        )}
      </WorkspaceLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
