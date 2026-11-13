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
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useModuleFilters } from "@/hooks/store/use-module-filters";
import { useModules } from "@/hooks/store/use-modules";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { ModuleAppliedFiltersList } from "../applied-filters/root";
import { ArchivedModulesView } from "./view";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * A project's archive of modules (P6a spec 2.3, 2.6, R3, R12): it reads the project's archived modules, a member's
 * (its guest's too) read of a project not archived (usePlanningReads' readable), whether its modules are turned on or
 * not: nerve reads them, and the archive shows them read only. It waits for nerve's first answer, says why nerve could
 * not give it, with a retry, or that the project has none yet; else the filters applied and the modules.
 */
export const ArchivedModuleLayoutRoot = observer(function ArchivedModuleLayoutRoot(props: Props) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getArchivedModuleIds, fetchArchivedModules } = useModules();
  const { getFilters } = useModuleFilters();
  const { readable } = usePlanningReads(workspaceSlug, projectId, "modules");
  const archive = useSessionSWR(readable ? ["PROJECT_ARCHIVED_MODULES", projectId] : null, (id) =>
    fetchArchivedModules(id)
  );
  const archivedModuleIds = getArchivedModuleIds(projectId);

  if (!readable) return null;

  if (archive.error !== undefined)
    return <PlanningReadFailure error={archive.error} onRetry={() => void archive.mutate()} />;

  if (archivedModuleIds === undefined) return <CycleModuleListLayoutLoader />;

  return (
    <>
      {calculateTotalFilters(getFilters(projectId, "archived")) !== 0 && (
        <div className="border-b border-subtle px-5 py-3">
          <ModuleAppliedFiltersList projectId={projectId} list="archived" />
        </div>
      )}
      {archivedModuleIds.length === 0 ? (
        <div className="h-full place-items-center">
          <EmptyStateDetailed
            assetKey="archived-module"
            title={t("workspace_empty_state.archive_modules.title")}
            description={t("workspace_empty_state.archive_modules.description")}
          />
        </div>
      ) : (
        <div className="relative h-full w-full overflow-auto">
          <ArchivedModulesView workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      )}
    </>
  );
});
