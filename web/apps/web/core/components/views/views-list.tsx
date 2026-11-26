/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { View } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
// components
import { ListLayout } from "@/components/core/list";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { viewsCreatableBy } from "@/lib/views-entry";
// local imports
import { useViewsPage } from "./actions-host";
import { useViewsPageState } from "./page-state";
import { ProjectViewListItem } from "./view-list-item";
import { viewsShown } from "./view-list-order";

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The project's views, as nerve lists them. */
  views: View[];
};

/**
 * The views of the project of projectId, at its page's address (its route's params, M3 handoff §4), as nerve lists them
 * (P6a spec 2.3, R4): none, how to make the first, by the page's dialog of a new view (ViewActionsHost), which only
 * those who may create a view may (viewsCreatableBy); else the views the page's search and filters keep, in the order
 * chosen (useViewsPageState, viewsShown), or, keeping none, that none matches.
 */
export const ProjectViewsList = observer(function ProjectViewsList(props: Props) {
  const { workspaceSlug, projectId, views } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { openCreate } = useViewsPage();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { choices } = useViewsPageState();
  // derived values
  const creatable = viewsCreatableBy(
    getProjectById(projectId),
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
  );

  if (views.length === 0)
    return (
      <EmptyStateDetailed
        assetKey="view"
        title={t("project_empty_state.views.title")}
        description={t("project_empty_state.views.description")}
        actions={[
          {
            label: t("project_empty_state.views.cta_primary"),
            onClick: openCreate,
            disabled: !creatable,
            variant: "primary",
          },
        ]}
      />
    );

  const shown = viewsShown(views, choices);
  if (shown.length === 0)
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
      />
    );

  return (
    <div className="flex h-full w-full flex-col">
      <ListLayout>
        {shown.map((view) => (
          <ProjectViewListItem key={view.id} view={view} workspaceSlug={workspaceSlug} projectId={projectId} />
        ))}
      </ListLayout>
    </div>
  );
});
