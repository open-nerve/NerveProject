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
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { Header, EHeaderVariant } from "@nerve/ui";
import { calculateTotalFilters } from "@nerve/utils";
// assets
import darkEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-dark.webp?url";
import lightEmptyState from "@/app/assets/empty-state/disabled-feature/cycles-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { CycleAppliedFiltersList } from "@/components/cycles/applied-filters";
import { CyclesView } from "@/components/cycles/cycles-view";
import { CyclesReadFailure } from "@/components/cycles/read-failure";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import type { Route } from "./+types/page";

/**
 * A project's cycles list (P5a spec 2.5, R8), of the route's project. Its cycles turned off, it says so, and only its
 * admins may go to its features to turn them on. Else it shows nerve's list of the project's cycles not archived (the
 * read the project's wrapper makes, which the page shares, use-project-fetch.ts): it waits for nerve's first answer,
 * says why nerve could not give it, with a retry, or how to set the project's first cycle (those who write its cycles
 * may, by the project's dialogs' host); else the filters applied and the cycles (CyclesView).
 */
function ProjectCyclesPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  // router
  const navigate = useNavigate();
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectCycleIds, fetchCycles } = useCycles();
  const { getFilters } = useCycleFilters();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { toggleCreateCycleModal } = useCommandPalette();
  // a member's read, of a project not archived whose cycles are on: as the project's wrapper reads them
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const listed = role !== undefined && project?.archived_at === null && project.cycle_view;
  const cycles = useSessionSWR(listed ? ["PROJECT_CYCLES", projectId] : null, (id) => fetchCycles(id));
  const cycleIds = getProjectCycleIds(projectId);

  if (!project) return null;

  const pageHead = <PageHead title={`${project.name} - ${t("common.cycles", { count: 2 })}`} />;

  if (!project.cycle_view)
    return (
      <div className="flex h-full w-full items-center justify-center">
        {pageHead}
        <DetailedEmptyState
          title={t("disabled_project.empty_state.cycle.title")}
          description={t("disabled_project.empty_state.cycle.description")}
          assetPath={resolvedTheme === "light" ? lightEmptyState : darkEmptyState}
          primaryButton={{
            text: t("disabled_project.empty_state.cycle.primary_button.text"),
            onClick: () => navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/cycles`),
            disabled: role !== EUserPermissions.ADMIN,
          }}
        />
      </div>
    );

  if (!listed) return null;

  if (cycles.error !== undefined)
    return <CyclesReadFailure error={cycles.error} onRetry={() => void cycles.mutate()} />;

  if (cycleIds === undefined) return <CycleModuleListLayoutLoader />;

  const canWrite = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  return (
    <>
      {pageHead}
      <div className="h-full w-full">
        {cycleIds.length === 0 ? (
          <div className="h-full place-items-center">
            <EmptyStateDetailed
              assetKey="cycle"
              title={t("project_empty_state.cycles.title")}
              description={t("project_empty_state.cycles.description")}
              actions={[
                {
                  label: t("project_empty_state.cycles.cta_primary"),
                  onClick: () => toggleCreateCycleModal(true),
                  variant: "primary",
                  disabled: !canWrite,
                },
              ]}
            />
          </div>
        ) : (
          <>
            {calculateTotalFilters(getFilters(projectId, "default")) !== 0 && (
              <Header variant={EHeaderVariant.TERNARY}>
                <CycleAppliedFiltersList projectId={projectId} list="default" />
              </Header>
            )}
            <CyclesView workspaceSlug={workspaceSlug} projectId={projectId} />
          </>
        )}
      </div>
    </>
  );
}

export default observer(ProjectCyclesPage);
