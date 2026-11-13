/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
import { cn } from "@nerve/utils";
// assets
import emptyModule from "@/app/assets/empty-state/module.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { ModuleDetailsSidebar } from "@/components/modules/progress-sidebar";
import { PlanningReadFailure } from "@/components/planning/read-failure";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";
import useLocalStorage from "@/hooks/use-local-storage";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { problemCode } from "@/lib/api-error";
import { useSessionSWR } from "@/lib/use-session-swr";
import type { Route } from "./+types/page";

/**
 * A module's page (P6a spec 2.6, R3, R12), of the route's project and module: its progress sidebar, on the right (its
 * work items come with P6b). It reads the module, its progress and its links for the caller who has a role in the
 * project, a guest too, while the project is not archived, whether the project's modules are turned on or not
 * (usePlanningReads' readable); the sidebar's controls and the header's quick actions offer no change while they are
 * off (canWriteModules). It says that nerve finds no such module in the project (deleted, or never there), with the
 * way back to its list; why nerve could not read one of them, with a retry; and waits for nerve's first answer, unless
 * the project's list read already gives the module. An archived module shows as any other, read-only.
 */
function ModuleDetailPage({ params }: Route.ComponentProps) {
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, moduleId } = params;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getModuleById, fetchModule, fetchProgress, fetchLinks } = useModules();
  const { getProjectById } = useProject();
  // hooks
  const { setValue, storedValue } = useLocalStorage("module_sidebar_collapsed", false);

  const project = getProjectById(projectId);
  const { readable } = usePlanningReads(workspaceSlug, projectId, "modules");
  const read = useSessionSWR(readable ? ["MODULE", moduleId] : null, (id) => fetchModule(id));
  const progress = useSessionSWR(readable ? ["MODULE_PROGRESS", moduleId] : null, (id) => fetchProgress(id));
  const links = useSessionSWR(readable ? ["MODULE_LINKS", moduleId] : null, (id) => fetchLinks(id));
  // the route's project's module alone: one of another project is not found here
  const held = getModuleById(moduleId);
  const module = held?.project_id === projectId ? held : undefined;

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => setValue(!isSidebarCollapsed);

  if (!readable) return null;

  const failed = [read, progress, links].filter((fetch) => fetch.error !== undefined);
  // nerve has no such module: it said so, it deleted it since, or it gave one of another project
  if (
    failed.some((fetch) => problemCode(fetch.error) === "planning.module_not_found") ||
    (module === undefined && read.data !== undefined)
  )
    return (
      <EmptyState
        image={emptyModule}
        title={t("project_module.not_found.title")}
        description={t("project_module.not_found.description")}
        primaryButton={{
          text: t("project_module.not_found.action"),
          onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/modules`),
        }}
      />
    );

  if (failed.length > 0)
    return (
      <PlanningReadFailure error={failed[0]?.error} onRetry={() => failed.forEach((fetch) => void fetch.mutate())} />
    );

  if (module === undefined)
    return (
      <Loader className="flex h-full flex-col items-end p-5">
        <Loader.Item height="30px" width="24rem" />
        <Loader.Item height="15px" width="24rem" />
      </Loader>
    );

  return (
    <>
      <PageHead title={project ? `${project.name} - ${module.name}` : undefined} />
      <div className="flex h-full w-full">
        {!isSidebarCollapsed && (
          <div
            className={cn(
              "vertical-scrollbar absolute right-0 z-13 scrollbar-sm flex h-full w-[24rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-6 shadow-raised-200 duration-300"
            )}
          >
            <ModuleDetailsSidebar
              handleClose={toggleSidebar}
              moduleId={moduleId}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
            />
          </div>
        )}
      </div>
    </>
  );
}

export default observer(ModuleDetailPage);
