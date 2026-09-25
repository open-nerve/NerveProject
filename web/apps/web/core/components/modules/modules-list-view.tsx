/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams, useSearchParams } from "react-router";
// components
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EUserProjectRoles } from "@nerve/types";
import { ContentWrapper, Row, ERowVariant } from "@nerve/ui";
import { ListLayout } from "@/components/core/list";
import { ModuleCardItem, ModuleListItem, ModulePeekOverview } from "@/components/modules";
import { CycleModuleBoardLayoutLoader } from "@/components/ui/loader/cycle-module-board-loader";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useModule } from "@/hooks/store/use-module";
import { useModuleFilter } from "@/hooks/store/use-module-filter";
import { useUserPermissions } from "@/hooks/store/user";

export const ModulesListView = observer(function ModulesListView() {
  // router
  const { workspaceSlug, projectId } = useParams();
  const [searchParams] = useSearchParams();
  const peekModule = searchParams.get("peekModule");
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateModuleModal } = useCommandPalette();
  const { getProjectModuleIds, getFilteredModuleIds, loader } = useModule();
  const { currentProjectDisplayFilters: displayFilters } = useModuleFilter();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const projectModuleIds = projectId ? getProjectModuleIds(projectId) : undefined;
  const filteredModuleIds = projectId ? getFilteredModuleIds(projectId) : undefined;
  const canPerformEmptyStateActions = allowPermissions(
    [EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER],
    EUserPermissionsLevel.PROJECT
  );

  if (loader || !projectModuleIds || !filteredModuleIds)
    return (
      <>
        {displayFilters?.layout === "list" && <CycleModuleListLayoutLoader />}
        {displayFilters?.layout === "board" && <CycleModuleBoardLayoutLoader />}
      </>
    );

  if (projectModuleIds.length === 0)
    return (
      <EmptyStateDetailed
        assetKey="module"
        title={t("project_empty_state.modules.title")}
        description={t("project_empty_state.modules.description")}
        actions={[
          {
            label: t("project_empty_state.modules.cta_primary"),
            onClick: () => toggleCreateModuleModal(true),
            disabled: !canPerformEmptyStateActions,
            variant: "primary",
          },
        ]}
      />
    );

  if (filteredModuleIds.length === 0)
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
      />
    );

  return (
    <ContentWrapper variant={ERowVariant.HUGGING}>
      <div className="flex size-full justify-between">
        {displayFilters?.layout === "list" && (
          <ListLayout>
            {filteredModuleIds.map((moduleId) => (
              <ModuleListItem key={moduleId} moduleId={moduleId} />
            ))}
          </ListLayout>
        )}
        {displayFilters?.layout === "board" && (
          <Row
            className={`grid size-full grid-cols-1 gap-6 overflow-y-auto py-page-y ${
              peekModule
                ? "3xl:grid-cols-3 lg:grid-cols-1 xl:grid-cols-2"
                : "3xl:grid-cols-4 lg:grid-cols-2 xl:grid-cols-3"
            } vertical-scrollbar scrollbar-lg auto-rows-max transition-all`}
          >
            {filteredModuleIds.map((moduleId) => (
              <ModuleCardItem key={moduleId} moduleId={moduleId} />
            ))}
          </Row>
        )}
        <div className="flex-shrink-0">
          <ModulePeekOverview projectId={projectId ?? ""} workspaceSlug={workspaceSlug ?? ""} />
        </div>
      </div>
    </ContentWrapper>
  );
});
