/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EIssuesStoreType, EUserProjectRoles } from "@nerve/types";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUserPermissions } from "@/hooks/store/user";
import { useWorkItemFilterInstance } from "@/hooks/store/work-item-filters/use-work-item-filter-instance";
import { useCycleListState } from "@/hooks/use-cycle-list-state";

/**
 * A cycle's page without work items to show (P5b spec 2.6): once the cycle has ended or is archived, only why; else the
 * clearing of the page's filters while any is applied; else a new work item of the cycle, for those who may create one
 * in the project (as the project page's) while the cycle takes them. Adding existing work items comes with P7 (R11).
 */
export const CycleEmptyState = observer(function CycleEmptyState() {
  // router
  const { workspaceSlug, projectId, cycleId } = useParams();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const { readOnly, joinable } = useCycleListState();
  const cycleWorkItemFilter = useWorkItemFilterInstance(EIssuesStoreType.CYCLE, cycleId);
  const canCreate =
    joinable &&
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );

  return (
    <div className="relative grid h-full w-full place-items-center overflow-y-auto">
      {readOnly ? (
        <EmptyStateDetailed
          assetKey="work-item"
          title={t("project_cycles.empty_state.completed_no_issues.title")}
          description={t("project_cycles.empty_state.completed_no_issues.description")}
        />
      ) : cycleWorkItemFilter?.hasActiveFilters ? (
        <EmptyStateDetailed
          assetKey="search"
          title={t("common_empty_state.search.title")}
          description={t("common_empty_state.search.description")}
          actions={[
            {
              label: t("project_issues.empty_state.issues_empty_filter.secondary_button.text"),
              onClick: cycleWorkItemFilter.clearFilters,
              variant: "secondary",
            },
          ]}
        />
      ) : (
        <EmptyStateDetailed
          assetKey="work-item"
          title={t("project_empty_state.cycle_work_items.title")}
          description={t("project_empty_state.cycle_work_items.description")}
          actions={
            canCreate
              ? [
                  {
                    label: t("project_empty_state.cycle_work_items.cta_primary"),
                    onClick: () => toggleCreateIssueModal(true, EIssuesStoreType.CYCLE),
                    variant: "primary",
                  },
                ]
              : []
          }
        />
      )}
    </div>
  );
});
