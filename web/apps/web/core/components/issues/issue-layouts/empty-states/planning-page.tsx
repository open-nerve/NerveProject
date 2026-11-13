/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { EUserProjectRoles } from "@nerve/types";
// hooks
import { PLANNING_PAGE_DOMAINS } from "@/hooks/planning-page-domains";
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUserPermissions } from "@/hooks/store/user";
import { useWorkItemFilterInstance } from "@/hooks/store/work-item-filters/use-work-item-filter-instance";
// local imports
import { PLANNING_PAGE_LAYOUTS, type PlanningPageLayoutStoreType } from "../planning-page-layouts";

/**
 * A planning page without work items to show (P5b spec 2.6, P6b spec 2.2), in its domain's words: once its list is
 * read-only (a cycle's: once the cycle has ended or is archived), only why; else the clearing of the page's filters
 * while any is applied; else a new work item of the page, for those who may create one in the project (as the project
 * page's) while the page's cycle takes them. Adding existing work items comes with P7 (R11).
 */
export const PlanningPageEmptyState = observer(function PlanningPageEmptyState(props: {
  storeType: PlanningPageLayoutStoreType;
}) {
  const { storeType } = props;
  // router
  const params = useParams();
  const { workspaceSlug, projectId } = params;
  const entityId = params[PLANNING_PAGE_DOMAINS[storeType].param];
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const { useListState, empty } = PLANNING_PAGE_LAYOUTS[storeType];
  const { readOnly, joinable } = useListState();
  const pageWorkItemFilter = useWorkItemFilterInstance(storeType, entityId);
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
          title={t(empty.readOnly.title)}
          description={t(empty.readOnly.description)}
        />
      ) : pageWorkItemFilter?.hasActiveFilters ? (
        <EmptyStateDetailed
          assetKey="search"
          title={t("common_empty_state.search.title")}
          description={t("common_empty_state.search.description")}
          actions={[
            {
              label: t("project_issues.empty_state.issues_empty_filter.secondary_button.text"),
              onClick: pageWorkItemFilter.clearFilters,
              variant: "secondary",
            },
          ]}
        />
      ) : (
        <EmptyStateDetailed
          assetKey="work-item"
          title={t(empty.none.title)}
          description={t(empty.none.description)}
          actions={
            canCreate
              ? [
                  {
                    label: t(empty.none.create),
                    onClick: () => toggleCreateIssueModal(true, storeType),
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
