/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EHeaderVariant, Header } from "@nerve/ui";
// assets
import darkViewsAsset from "@/app/assets/empty-state/disabled-feature/views-dark.webp?url";
import lightViewsAsset from "@/app/assets/empty-state/disabled-feature/views-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { ViewListLoader } from "@/components/ui/loader/view-list-loader";
import { ViewAppliedFiltersList } from "@/components/views/applied-filters";
import { useViewsPageState } from "@/components/views/page-state";
import { ProjectViewsList } from "@/components/views/views-list";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import { viewEntryOn } from "@/lib/views-entry";
import type { Route } from "./+types/page";

/**
 * A project's views list (P6a spec 2.2, 2.3, R4, R20), of the route's project. Its views turned off, it says so, reads
 * none, and only its admins may go to its features to turn them on. Else, for those the ways into its views are offered
 * to (viewEntryOn), it reads nerve's list of them (the one read of a project's views, ["PROJECT_VIEWS", projectId],
 * which the views store keeps): it waits for nerve's first answer, says why nerve could not give it, with a retry; else
 * the filters applied to the list (the page's, useViewsPageState) and the views (ProjectViewsList).
 */
function ProjectViewsPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  // router
  const navigate = useNavigate();
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { getProjectViews, fetchProjectViews } = useViews();
  const { choices, setChoices } = useViewsPageState();
  // derived values: of the address's project, never the router store's (M3 handoff §4)
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const entry = viewEntryOn(project, role);
  const views = useSessionSWR(entry ? ["PROJECT_VIEWS", projectId] : null, (id) => fetchProjectViews(id));
  const listed = getProjectViews(projectId);

  if (!project) return null;

  const pageHead = <PageHead title={`${project.name} - ${t("views")}`} />;

  if (!project.issue_views_view)
    return (
      <>
        {pageHead}
        <div className="flex h-full w-full items-center justify-center">
          <DetailedEmptyState
            title={t("disabled_project.empty_state.view.title")}
            description={t("disabled_project.empty_state.view.description")}
            assetPath={resolvedTheme === "light" ? lightViewsAsset : darkViewsAsset}
            primaryButton={{
              text: t("disabled_project.empty_state.view.primary_button.text"),
              onClick: () => navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/views`),
              disabled: role !== EUserPermissions.ADMIN,
            }}
          />
        </div>
      </>
    );

  if (!entry) return null;

  if (views.error !== undefined) return <PlanningReadFailure error={views.error} onRetry={() => void views.mutate()} />;

  if (listed === undefined) return <ViewListLoader />;

  const filtered = choices.filters.owned_by.length > 0 || choices.filters.created_at.length > 0;
  return (
    <>
      {pageHead}
      {listed.length > 0 && filtered && (
        <Header variant={EHeaderVariant.TERNARY}>
          <ViewAppliedFiltersList
            filters={choices.filters}
            onChange={(filters) => setChoices((current) => ({ ...current, filters }))}
          />
        </Header>
      )}
      <ProjectViewsList workspaceSlug={workspaceSlug} projectId={projectId} views={listed} />
    </>
  );
}

export default observer(ProjectViewsPage);
