/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { ModuleListItem as ModuleRow } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { ContentWrapper, Row, ERowVariant } from "@nerve/ui";
import type { ModuleFilters } from "@nerve/utils";
import { cn, matchesModuleFilters, orderModuleRows } from "@nerve/utils";
// assets
import AllFiltersImage from "@/app/assets/empty-state/module/all-filters.svg?url";
import NameFilterImage from "@/app/assets/empty-state/module/name-filter.svg?url";
// components
import { ListLayout } from "@/components/core/list";
// hooks
import { useModuleFilters } from "@/hooks/store/use-module-filters";
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { ModuleCardItem } from "./module-card-item";
import { ModuleListItem } from "./module-list-item";
import { ModulePeekOverview, usePeekModule } from "./module-peek-overview";
import { useModuleDisplay } from "./use-module-display";

/**
 * The rows of moduleIds the store gives that pass filters and whose names hold query, whatever its case, in the order
 * of moduleIds.
 */
function filterModuleRows(
  moduleIds: string[],
  rowOf: (moduleId: string) => ModuleRow | undefined,
  filters: ModuleFilters,
  query: string
): ModuleRow[] {
  const searched = query.trim().toLowerCase();
  return moduleIds.flatMap((moduleId) => {
    const row = rowOf(moduleId);
    return row && matchesModuleFilters(row, filters) && row.name.toLowerCase().includes(searched) ? [row] : [];
  });
}

/** That none of a project's modules match its filters and its search, and which to clear to see them all. */
function NoMatchingModules({ searched }: { searched: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="grid h-full w-full place-items-center">
      <div className="text-center">
        <img
          src={searched ? NameFilterImage : AllFiltersImage}
          className="mx-auto h-36 w-36 object-contain sm:h-48 sm:w-48"
          alt=""
        />
        <h5 className="mt-7 mb-1 text-18 font-medium">{t("project_module.no_matching_modules")}</h5>
        <p className="text-14 text-placeholder">
          {searched
            ? t("project_module.remove_search_criteria_to_see_all_modules")
            : t("project_module.remove_filters_to_see_all_modules")}
        </p>
      </div>
    </div>
  );
}

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * A project's modules list (P6a spec 2.4, 2.6, R4, R14), of its modules not archived as nerve lists them, filtered and
 * searched by the project's own filters and search (the session's, ModuleFilterStore), ordered and laid out as the
 * person chose for the project (the browser's, useModuleDisplay): rows, or cards; and the preview of the module
 * ?peekModule names. Nothing matching, it says which to clear.
 */
export const ModulesListView = observer(function ModulesListView(props: Props) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const { getProjectModuleIds, getModuleRow } = useModules();
  const { getFilters, getSearchQuery } = useModuleFilters();
  const { layout, orderBy } = useModuleDisplay(projectId);
  const [peekModule] = usePeekModule();
  // derived values
  const query = getSearchQuery(projectId, "default");
  const rows = orderModuleRows(
    filterModuleRows(getProjectModuleIds(projectId) ?? [], getModuleRow, getFilters(projectId, "default"), query),
    orderBy
  );

  if (rows.length === 0) return <NoMatchingModules searched={query.trim() !== ""} />;

  return (
    <ContentWrapper variant={ERowVariant.HUGGING}>
      <div className="flex size-full justify-between">
        {layout === "list" ? (
          <ListLayout>
            {rows.map((row) => (
              <ModuleListItem key={row.id} moduleId={row.id} workspaceSlug={workspaceSlug} projectId={projectId} />
            ))}
          </ListLayout>
        ) : (
          <Row
            className={cn(
              "vertical-scrollbar scrollbar-lg grid size-full auto-rows-max grid-cols-1 gap-6 overflow-y-auto py-page-y transition-all",
              peekModule
                ? "3xl:grid-cols-3 lg:grid-cols-1 xl:grid-cols-2"
                : "3xl:grid-cols-4 lg:grid-cols-2 xl:grid-cols-3"
            )}
          >
            {rows.map((row) => (
              <ModuleCardItem key={row.id} moduleId={row.id} workspaceSlug={workspaceSlug} projectId={projectId} />
            ))}
          </Row>
        )}
        <div className="flex-shrink-0">
          <ModulePeekOverview workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      </div>
    </ContentWrapper>
  );
});
