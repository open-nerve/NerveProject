/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EUserProjectRoles } from "@nerve/types";
// components
import { ListLayout } from "@/components/core/list";
import { ViewListLoader } from "@/components/ui/loader/view-list-loader";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProjectView } from "@/hooks/store/use-project-view";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { ProjectViewListItem } from "./view-list-item";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The views of the project of projectId, at its page's address (its route's params, M3 handoff §4): what the caller may
 * do there follows his role in that project, never in the project the router store holds.
 */
export const ProjectViewsList = observer(function ProjectViewsList(props: Props) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateViewModal } = useCommandPalette();
  const { getProjectViews, getFilteredProjectViews, loader } = useProjectView();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const projectViews = getProjectViews(projectId);
  const filteredProjectViews = getFilteredProjectViews(projectId);
  const canPerformEmptyStateActions = allowPermissions(
    [EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER, EUserProjectRoles.GUEST],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  if (loader || !projectViews || !filteredProjectViews) return <ViewListLoader />;

  if (filteredProjectViews.length === 0 && projectViews.length > 0) {
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
      />
    );
  }

  return (
    <>
      {filteredProjectViews.length > 0 ? (
        <div className="flex h-full w-full flex-col">
          <ListLayout>
            {filteredProjectViews.length > 0 ? (
              filteredProjectViews.map((view) => <ProjectViewListItem key={view.id} view={view} />)
            ) : (
              <p className="mt-10 text-center text-13 text-tertiary">No results found</p>
            )}
          </ListLayout>
        </div>
      ) : (
        <EmptyStateDetailed
          assetKey="view"
          title={t("project_empty_state.views.title")}
          description={t("project_empty_state.views.description")}
          actions={[
            {
              label: t("project_empty_state.views.cta_primary"),
              onClick: () => toggleCreateViewModal(true),
              disabled: !canPerformEmptyStateActions,
              variant: "primary",
            },
          ]}
        />
      )}
    </>
  );
});
