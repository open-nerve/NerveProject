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
import { Loader } from "@nerve/ui";
// assets
import darkViewsAsset from "@/app/assets/empty-state/disabled-feature/views-dark.webp?url";
import lightViewsAsset from "@/app/assets/empty-state/disabled-feature/views-light.webp?url";
import emptyView from "@/app/assets/empty-state/view.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { ProjectViewLayoutRoot } from "@/components/issues/issue-layouts/roots/project-view-layout-root";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { useViewPageRead } from "@/components/views/use-view-page-read";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useViews } from "@/hooks/store/use-views";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";
import { viewEntryOn } from "@/lib/views-entry";
import type { Route } from "./+types/page";

/**
 * A project's view's page (P6a spec 2.2, 2.6, R6), of the route's project and view (its layout is keyed by the view).
 * Its views turned off, it says so, reads nothing, and only its admins may go to its features to turn them on. Else,
 * for those the ways into the project's views are offered to (viewEntryOn), the view's visit: each session's its own.
 */
function ProjectViewIssuesPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId, viewId } = params;
  // router
  const navigate = useNavigate();
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  const { loginId } = useSession();
  // store hooks
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // derived values: of the address's project, never the router store's (M3 handoff §4)
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);

  if (!project) return null;

  if (!project.issue_views_view)
    return (
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
    );

  if (!viewEntryOn(project, role)) return null;

  return (
    <ViewVisit
      key={loginId}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      viewId={viewId}
      projectName={project.name}
    />
  );
}

type VisitProps = { workspaceSlug: string; projectId: string; viewId: string; projectName: string };

/**
 * One visit of the view's page, in one session (a new session's stores are new: its visit is a new one), whose read of
 * the view is its own (useViewPageRead): it waits for nerve's answer; says that nerve finds no such view in the project
 * (deleted, hidden from the caller, never there, or of another project), with the way back to the project's views; why
 * nerve could not read it, with a retry; shows nothing once the read says the page is gone; and else the project's work
 * items under the page's draft of the view (ProjectViewLayoutRoot).
 */
const ViewVisit = observer(function ViewVisit(props: VisitProps) {
  const { workspaceSlug, projectId, viewId, projectName } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getViewById } = useViews();
  const read = useViewPageRead(workspaceSlug, viewId);
  // the route's project's view alone: one of another project is not found here
  const held = getViewById(viewId);
  const view = held?.project_id === projectId ? held : undefined;

  if (read.state === "reading")
    return (
      <Loader className="flex h-full flex-col p-5">
        <Loader.Item height="30px" width="100%" />
        <Loader.Item height="30px" width="100%" />
      </Loader>
    );
  if (read.state === "gone") return null;
  if (read.state === "failed") return <PlanningReadFailure error={read.error} onRetry={read.retry} />;
  // nerve has no such view in the project: it said so, or gave one of another project
  if (read.state === "not-found" || !view)
    return (
      <EmptyState
        image={emptyView}
        title={t("project_views.not_found.title")}
        description={t("project_views.not_found.description")}
        primaryButton={{
          text: t("project_views.not_found.action"),
          onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/views`),
        }}
      />
    );

  return (
    <>
      <PageHead title={`${projectName} - ${view.name}`} />
      <div className="h-full w-full">
        <ProjectViewLayoutRoot workspaceSlug={workspaceSlug} projectId={projectId} viewId={viewId} />
      </div>
    </>
  );
});

export default observer(ProjectViewIssuesPage);
