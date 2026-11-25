/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
// nerve imports
import { EIssuesStoreType } from "@nerve/types";
// components
import { MobileHeaderFilters } from "@/components/issues/mobile-header-filters";
import { useShownView } from "@/components/views/use-view-page-read";
// hooks
import { useViews } from "@/hooks/store/use-views";

type Props = {
  /** The page's address, as the route gives it. */
  workspaceSlug: string;
  projectId: string;
  viewId: string;
};

/**
 * The small screen's header of a project's view's page (P6a spec 2.6; re-96 m8): the layouts and display of the page's
 * draft of the view (MobileHeaderFilters), as the header at a wider width gives its controls, while the page shows the
 * view (useShownView): not while it reads it, finds none, could not read it, or the project's views are off; stopped
 * while a change of the view is out (R29). An
 * observer, so that the page's read answered after the page opened shows them: the layout that passes it is not one
 * (re-76 F1), as PlanningPageMobileHeader.
 */
export const ProjectViewMobileHeader = observer(function ProjectViewMobileHeader(props: Props) {
  const { workspaceSlug, projectId, viewId } = props;
  const { view } = useShownView(workspaceSlug, projectId, viewId);
  const { isUpdating } = useViews();

  if (!view) return null;

  return (
    <MobileHeaderFilters storeType={EIssuesStoreType.PROJECT_VIEW} entityId={viewId} disabled={isUpdating(viewId)} />
  );
});
