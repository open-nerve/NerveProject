/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import useSWR from "swr";
// nerve imports
import { cn } from "@nerve/utils";
// assets
import emptyCycle from "@/app/assets/empty-state/cycle.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
import useCyclesDetails from "@/components/cycles/active-cycle/use-cycles-details";
import { CycleDetailsSidebar } from "@/components/cycles/progress-sidebar";
// hooks
import { useCycle } from "@/hooks/store/use-cycle";
import { useProject } from "@/hooks/store/use-project";
import { useNavigate } from "react-router";
import useLocalStorage from "@/hooks/use-local-storage";
import type { Route } from "./+types/page";

function CycleDetailPage({ params }: Route.ComponentProps) {
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, cycleId } = params;
  // store hooks
  const { getCycleById, loader, fetchCycleDetails } = useCycle();
  const { getProjectById } = useProject();
  // hooks
  const { setValue, storedValue } = useLocalStorage("cycle_sidebar_collapsed", false);

  useCyclesDetails({
    workspaceSlug,
    projectId,
    cycleId,
  });
  const { isLoading: isCycleLoading } = useSWR(
    workspaceSlug && projectId && cycleId ? ["CYCLE_DETAILS", workspaceSlug, projectId, cycleId] : null,
    workspaceSlug && projectId && cycleId ? () => fetchCycleDetails(workspaceSlug, projectId, cycleId) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // derived values
  const isSidebarCollapsed = storedValue === true;
  const cycle = getCycleById(cycleId);
  const project = getProjectById(projectId);
  const pageTitle = project?.name && cycle?.name ? `${project?.name} - ${cycle?.name}` : undefined;

  /**
   * Toggles the sidebar
   */
  const toggleSidebar = () => setValue(!isSidebarCollapsed);

  return (
    <>
      <PageHead title={pageTitle} />
      {!cycle && !loader && !isCycleLoading ? (
        <EmptyState
          image={emptyCycle}
          title="Cycle does not exist"
          description="The cycle you are looking for does not exist or has been deleted."
          primaryButton={{
            text: "View other cycles",
            onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/cycles`),
          }}
        />
      ) : (
        <>
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
      )}
    </>
  );
}

export default observer(CycleDetailPage);
