/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EHeaderVariant, Header } from "@nerve/ui";
import { calculateTotalFilters } from "@nerve/utils";
// components
import { PageHead } from "@/components/core/page-title";
import { ModuleAppliedFiltersList } from "@/components/modules/applied-filters/root";
import { ModulesListView } from "@/components/modules/modules-list-view";
import { useModuleDisplay } from "@/components/modules/use-module-display";
import { PlanningDisabled } from "@/components/planning/disabled";
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { CycleModuleBoardLayoutLoader } from "@/components/ui/loader/cycle-module-board-loader";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useModuleFilters } from "@/hooks/store/use-module-filters";
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { planningOn } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";
import type { Route } from "./+types/page";

/**
 * A project's modules list (P6a spec 2.6, R3, R4), of the route's project. Its modules turned off, it says so, and only
 * its admins may go to its features to turn them on. Else it shows nerve's list of the project's modules not archived
 * (the read the project's wrapper makes, which the page shares, use-project-fetch.ts): it waits for nerve's first
 * answer as the list is laid out (rows or cards, useModuleDisplay), says why nerve could not give it, with a retry, or
 * how to set the project's first module (those who write its modules may, canWriteModules, by the project's dialogs'
 * host); else the filters applied to the list (the session's, ModuleFilterStore) and the modules (ModulesListView).
 */
function ProjectModulesPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectModuleIds, fetchModules } = useModules();
  const { getFilters } = useModuleFilters();
  const { getProjectById } = useProject();
  const { canWriteModules } = useUserPermissions();
  const { toggleCreateModuleModal } = useCommandPalette();
  const { layout } = useModuleDisplay(projectId);
  const project = getProjectById(projectId);
  const { listed } = usePlanningReads(workspaceSlug, projectId, "modules");
  const modules = useSessionSWR(listed ? ["PROJECT_MODULES", projectId] : null, (id) => fetchModules(id));
  const moduleIds = getProjectModuleIds(projectId);

  if (!project) return null;

  const pageHead = <PageHead title={`${project.name} - ${t("modules")}`} />;

  if (!planningOn(project, "modules"))
    return (
      <>
        {pageHead}
        <PlanningDisabled workspaceSlug={workspaceSlug} projectId={projectId} domain="modules" />
      </>
    );

  if (!listed) return null;

  if (modules.error !== undefined)
    return <PlanningReadFailure error={modules.error} onRetry={() => void modules.mutate()} />;

  if (moduleIds === undefined)
    return layout === "board" ? <CycleModuleBoardLayoutLoader /> : <CycleModuleListLayoutLoader />;

  return (
    <>
      {pageHead}
      <div className="flex h-full w-full flex-col">
        {moduleIds.length === 0 ? (
          <EmptyStateDetailed
            assetKey="module"
            title={t("project_empty_state.modules.title")}
            description={t("project_empty_state.modules.description")}
            actions={[
              {
                label: t("project_empty_state.modules.cta_primary"),
                onClick: () => toggleCreateModuleModal(true),
                disabled: !canWriteModules(workspaceSlug, projectId),
                variant: "primary",
              },
            ]}
          />
        ) : (
          <>
            {calculateTotalFilters(getFilters(projectId, "default")) !== 0 && (
              <Header variant={EHeaderVariant.TERNARY}>
                <ModuleAppliedFiltersList projectId={projectId} list="default" />
              </Header>
            )}
            <ModulesListView workspaceSlug={workspaceSlug} projectId={projectId} />
          </>
        )}
      </div>
    </>
  );
}

export default observer(ProjectModulesPage);
