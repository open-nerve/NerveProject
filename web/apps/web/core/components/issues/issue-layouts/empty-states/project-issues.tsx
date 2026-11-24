/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EIssuesStoreType, EUserProjectRoles } from "@nerve/types";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUserPermissions } from "@/hooks/store/user";
import { useWorkItemFilterInstance } from "@/hooks/store/work-item-filters/use-work-item-filter-instance";

/**
 * A list page of the project's work items with none to show, by the store type and id of the page's work item filter:
 * the project's page's, by the project; a project's view's page's, by the view (P6a spec 2.5). While the page's filter
 * applies any, the clearing of it; else a new work item, of the page, for the project's admins and members.
 */
export const ProjectListEmptyState = observer(function ProjectListEmptyState(props: {
  storeType: EIssuesStoreType.PROJECT | EIssuesStoreType.PROJECT_VIEW;
  filterId: string | undefined;
}) {
  const { storeType, filterId } = props;
  // router
  const { workspaceSlug, projectId } = useParams();
  // nerve imports
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // derived values
  const projectWorkItemFilter = useWorkItemFilterInstance(storeType, filterId);
  const canReadProject =
    !!workspaceSlug && !!projectId && getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined;

  const canPerformEmptyStateActions =
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );

  return (
    <div className="relative h-full w-full overflow-y-auto">
      {projectWorkItemFilter?.hasActiveFilters ? (
        <EmptyStateDetailed
          assetKey="search"
          title={t("common_empty_state.search.title")}
          description={t("common_empty_state.search.description")}
          actions={[
            {
              label: t("project_issues.empty_state.issues_empty_filter.secondary_button.text"),
              onClick: projectWorkItemFilter?.clearFilters,
              disabled: !canReadProject || !projectWorkItemFilter,
              variant: "secondary",
            },
          ]}
        />
      ) : (
        <EmptyStateDetailed
          assetKey="work-item"
          title={t("project_empty_state.work_items.title")}
          description={t("project_empty_state.work_items.description")}
          actions={[
            {
              label: t("project_empty_state.work_items.cta_primary"),
              onClick: () => {
                toggleCreateIssueModal(true, storeType);
              },
              disabled: !canPerformEmptyStateActions,
              variant: "primary",
            },
          ]}
        />
      )}
    </div>
  );
});

/** The project's page's, by the route's project. */
export function ProjectEmptyState() {
  const { projectId } = useParams();
  return <ProjectListEmptyState storeType={EIssuesStoreType.PROJECT} filterId={projectId} />;
}
