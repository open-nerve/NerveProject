/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Loader } from "@nerve/ui";
import { cn } from "@nerve/utils";
// assets
import emptyCycle from "@/app/assets/empty-state/cycle.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import { CycleDetailsSidebar } from "@/components/cycles/progress-sidebar";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import useLocalStorage from "@/hooks/use-local-storage";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { problemCode } from "@/lib/api-error";
import { useSessionSWR } from "@/lib/use-session-swr";
import type { Route } from "./+types/page";

/**
 * A cycle's page (P5a spec 2.2, 2.5, R11), of the route's project and cycle: until P5b brings its work items, its
 * progress sidebar. It reads the cycle and its progress, and for an archived cycle its project's archive, whose row
 * holds its members; a member's reads of a project not archived alone, as the project's own (use-project-fetch.ts).
 * It says that nerve finds no such cycle in the project (deleted, or never there), with the way back to its list; why
 * nerve could not read it, with a retry; and waits for nerve's first answer. An archived cycle shows as any other,
 * read-only: its quick actions and its dates offer no change of it.
 */
function CycleDetailPage({ params }: Route.ComponentProps) {
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, cycleId } = params;
  // nerve hooks
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  // store hooks
  const { getCycleById, fetchCycle, fetchProgress, fetchArchivedCycles } = useCycles();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // hooks
  const { setValue, storedValue } = useLocalStorage("cycle_sidebar_collapsed", false);

  // a member's reads, of a project not archived: nerve's read of the project decided both (ProjectAuthWrapper)
  const project = getProjectById(projectId);
  const readable =
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined && project?.archived_at === null;
  const read = useSessionSWR(readable ? ["CYCLE", cycleId] : null, (id) => fetchCycle(id));
  const progress = useSessionSWR(readable ? ["CYCLE_PROGRESS", cycleId] : null, (id) => fetchProgress(id));
  // the route's project's cycle alone: one of another project is not found here
  const held = getCycleById(cycleId);
  const cycle = held?.project_id === projectId ? held : undefined;
  const archive = readable && cycle !== undefined && cycle.archived_at !== null ? projectId : undefined;
  const archived = useSessionSWR(archive ? ["PROJECT_ARCHIVED_CYCLES", archive] : null, (id) =>
    fetchArchivedCycles(id)
  );

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => setValue(!isSidebarCollapsed);

  if (!readable) return null;

  const failed = [read, progress, archived].filter((fetch) => fetch.error !== undefined);
  // nerve has no such cycle: it said so, it deleted it since, or it gave one of another project
  if (
    failed.some((fetch) => problemCode(fetch.error) === "planning.cycle_not_found") ||
    (cycle === undefined && read.data !== undefined)
  )
    return (
      <EmptyState
        image={emptyCycle}
        title={t("project_cycles.not_found.title")}
        description={t("project_cycles.not_found.description")}
        primaryButton={{
          text: t("project_cycles.not_found.action"),
          onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/cycles`),
        }}
      />
    );

  if (failed.length > 0)
    return (
      <div role="alert" className="flex size-full flex-col items-center justify-center gap-3 text-13 text-secondary">
        <span>{refusalMessage(failed[0]?.error)}</span>
        <Button variant="secondary" size="sm" onClick={() => failed.forEach((fetch) => void fetch.mutate())}>
          {t("common.retry")}
        </Button>
      </div>
    );

  if (cycle === undefined)
    return (
      <Loader className="flex h-full flex-col items-end p-5">
        <Loader.Item height="30px" width="21.5rem" />
        <Loader.Item height="15px" width="21.5rem" />
      </Loader>
    );

  return (
    <>
      <PageHead title={project ? `${project.name} - ${cycle.name}` : undefined} />
      <div className="flex h-full w-full">
        {!isSidebarCollapsed && (
          <div
            className={cn(
              "vertical-scrollbar absolute right-0 z-13 scrollbar-sm flex h-full w-[21.5rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-4 shadow-raised-200 duration-300"
            )}
          >
            <CycleDetailsSidebar
              handleClose={toggleSidebar}
              cycleId={cycleId}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
            />
          </div>
        )}
      </div>
    </>
  );
}

export default observer(CycleDetailPage);
