/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { calculateTotalFilters } from "@nerve/utils";
// components
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { CycleAppliedFiltersList } from "../applied-filters";
import { CyclesReadFailure } from "../read-failure";
import { ArchivedCyclesView } from "./view";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * A project's archive of cycles (P5a spec 2.2, 2.5, R11): it reads the project's archived cycles, a member's read of a
 * project not archived alone (as the project's own, use-project-fetch.ts), and the read a cycle's page makes of an
 * archived cycle's archive too, which the two share. It waits for nerve's first answer, says why nerve could not give
 * it, with a retry, or that the project has none yet; else the filters applied and the cycles.
 */
export const ArchivedCycleLayoutRoot = observer(function ArchivedCycleLayoutRoot(props: Props) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getArchivedCycleIds, fetchArchivedCycles } = useCycles();
  const { getFilters } = useCycleFilters();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // a member's read, of a project not archived: nerve's read of the project decided both (ProjectAuthWrapper)
  const readable =
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined &&
    getProjectById(projectId)?.archived_at === null;
  const archive = useSessionSWR(readable ? ["PROJECT_ARCHIVED_CYCLES", projectId] : null, (id) =>
    fetchArchivedCycles(id)
  );
  const archivedCycleIds = getArchivedCycleIds(projectId);

  if (!readable) return null;

  if (archive.error !== undefined)
    return <CyclesReadFailure error={archive.error} onRetry={() => void archive.mutate()} />;

  if (archivedCycleIds === undefined) return <CycleModuleListLayoutLoader />;

  return (
    <>
      {calculateTotalFilters(getFilters(projectId, "archived")) !== 0 && (
        <div className="border-b border-subtle px-5 py-3">
          <CycleAppliedFiltersList projectId={projectId} list="archived" />
        </div>
      )}
      {archivedCycleIds.length === 0 ? (
        <div className="h-full place-items-center">
          <EmptyStateDetailed
            assetKey="archived-cycle"
            title={t("workspace_empty_state.archive_cycles.title")}
            description={t("workspace_empty_state.archive_cycles.description")}
          />
        </div>
      ) : (
        <div className="relative h-full w-full overflow-auto">
          <ArchivedCyclesView workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      )}
    </>
  );
});
