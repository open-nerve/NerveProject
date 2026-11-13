/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { orderModuleRows } from "@nerve/utils";
// hooks
import { useModuleFilters } from "@/hooks/store/use-module-filters";
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { ModuleListItem } from "../module-list-item";
import { ModulePeekOverview } from "../module-peek-overview";
import { NoMatchingModules, filterModuleRows } from "../modules-list-view";
import { useModuleDisplay } from "../use-module-display";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * A project's archived modules (P6a spec 2.4, 2.6, R4, R12), filtered and searched by the project's own filters and
 * search of its archive (the session's), ordered as the project's list is (the browser's one order of the project,
 * useModuleDisplay); each row opens the module's preview, its quick actions restore it. Nothing matching, it says
 * which to clear.
 */
export const ArchivedModulesView = observer(function ArchivedModulesView(props: Props) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const { getArchivedModuleIds, getModuleRow } = useModules();
  const { getFilters, getSearchQuery } = useModuleFilters();
  const { orderBy } = useModuleDisplay(projectId);
  // derived values
  const query = getSearchQuery(projectId, "archived");
  const rows = orderModuleRows(
    filterModuleRows(getArchivedModuleIds(projectId) ?? [], getModuleRow, getFilters(projectId, "archived"), query),
    orderBy
  );

  if (rows.length === 0) return <NoMatchingModules searched={query.trim() !== ""} />;

  return (
    <div className="h-full overflow-y-auto">
      <div className="flex h-full w-full justify-between">
        <div className="vertical-scrollbar scrollbar-lg flex h-full w-full flex-col overflow-y-auto">
          {rows.map((row) => (
            <ModuleListItem key={row.id} moduleId={row.id} workspaceSlug={workspaceSlug} projectId={projectId} />
          ))}
        </div>
        <ModulePeekOverview projectId={projectId} workspaceSlug={workspaceSlug} />
      </div>
    </div>
  );
});
