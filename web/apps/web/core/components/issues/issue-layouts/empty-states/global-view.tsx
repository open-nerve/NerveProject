/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useContext } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
import { useWorkItemFilterInstance } from "@/hooks/store/work-item-filters/use-work-item-filter-instance";
// lib
import { joinedAndUnarchived } from "@/lib/project-scope";
import { workspaceViewsCreatableBy } from "@/lib/views-entry";
// store
import { workspacePageId } from "@/store/issue/workspace/page-id";
// local imports
import { DraftBusyContext } from "../draft-busy";

/**
 * A workspace's view's page with no work items to show (M7/P6b spec 2.8, R24), of the route's workspace and view: while
 * the page's filter applies any, the clearing of it (none while the page's draft is being saved, R29); else, when the
 * caller is an active member of no unarchived project of the workspace, a new project; else a new work item of the
 * page. Its two creations are for the workspace's admins and members (nerve's project.create and work item rules):
 * its guest is offered none.
 */
export const GlobalViewEmptyState = observer(function GlobalViewEmptyState() {
  // router
  const { workspaceSlug = "", globalViewId } = useParams();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateIssueModal, toggleCreateProjectModal } = useCommandPalette();
  const busy = useContext(DraftBusyContext);
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { getWorkspaceBySlug } = useWorkspace();
  const { knownProjectIds, getProjectById } = useProject();
  // derived values
  const pageFilter = useWorkItemFilterInstance(
    EIssuesStoreType.GLOBAL,
    globalViewId ? workspacePageId(workspaceSlug, globalViewId) : undefined
  );
  const creates = workspaceViewsCreatableBy(getWorkspaceRoleByWorkspaceSlug(workspaceSlug));
  const workspace = getWorkspaceBySlug(workspaceSlug);
  const hasProjects =
    !!workspace &&
    knownProjectIds(workspace.id).some((id) => {
      const project = getProjectById(id);
      return !!project && joinedAndUnarchived(project);
    });

  if (pageFilter?.hasActiveFilters)
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
        actions={[
          {
            label: t("project_issues.empty_state.issues_empty_filter.secondary_button.text"),
            onClick: pageFilter.clearFilters,
            disabled: busy,
            variant: "secondary",
          },
        ]}
      />
    );

  if (!hasProjects)
    return (
      <EmptyStateDetailed
        title={t("workspace_projects.empty_state.no_projects.title")}
        description={t("workspace_projects.empty_state.no_projects.description")}
        assetKey="project"
        assetClassName="size-40"
        actions={
          creates
            ? [
                {
                  label: t("workspace_projects.empty_state.no_projects.primary_button.text"),
                  onClick: () => toggleCreateProjectModal(true),
                  variant: "primary",
                },
              ]
            : []
        }
      />
    );

  return (
    <EmptyStateDetailed
      title={t("workspace_empty_state.views.title")}
      description={t("workspace_empty_state.views.description")}
      assetKey="project"
      assetClassName="size-40"
      actions={
        creates
          ? [
              {
                label: t("workspace_empty_state.views.cta_primary"),
                onClick: () => toggleCreateIssueModal(true, EIssuesStoreType.GLOBAL),
                variant: "primary",
              },
            ]
          : []
      }
    />
  );
});
