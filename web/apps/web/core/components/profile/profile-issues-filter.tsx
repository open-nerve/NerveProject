/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { useLocalIssueFilterUpdate } from "@/hooks/use-local-issue-filter-update";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve constants
import { EIssueFilterType, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
// i18n
import { useTranslation } from "@nerve/i18n";
// types
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@nerve/types";
// components
import { DisplayFiltersSelection, FiltersDropdown, LayoutSelection } from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useIssues } from "@/hooks/store/use-issues";

export const ProfileIssuesFilter = observer(function ProfileIssuesFilter() {
  // i18n
  const { t } = useTranslation();
  // router
  const { workspaceSlug, userId } = useParams();
  // store hook
  const {
    issuesFilter: { getIssueFilters, updateFilters },
  } = useIssues(EIssuesStoreType.PROFILE);
  const issueFilters = workspaceSlug && userId ? getIssueFilters(userId, workspaceSlug) : undefined;
  const follow = useLocalIssueFilterUpdate(workspaceSlug ?? "", undefined, userId ?? "", true);
  // derived values
  const activeLayout = issueFilters?.displayFilters?.layout;

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!workspaceSlug || !userId) return;
      return follow(() =>
        updateFilters(workspaceSlug, undefined, EIssueFilterType.DISPLAY_FILTERS, { layout }, userId)
      );
    },
    [workspaceSlug, updateFilters, userId, follow]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      if (!workspaceSlug || !userId) return;
      return follow(() =>
        updateFilters(workspaceSlug, undefined, EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter, userId)
      );
    },
    [workspaceSlug, updateFilters, userId, follow]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      if (!workspaceSlug || !userId) return;
      return follow(() =>
        updateFilters(workspaceSlug, undefined, EIssueFilterType.DISPLAY_PROPERTIES, property, userId)
      );
    },
    [workspaceSlug, updateFilters, userId, follow]
  );

  return (
    <div className="relative flex items-center justify-end gap-2">
      <LayoutSelection
        layouts={[EIssueLayoutTypes.LIST, EIssueLayoutTypes.KANBAN]}
        onChange={(layout) => handleLayoutChange(layout)}
        selectedLayout={activeLayout}
      />
      {userId && <WorkItemFiltersToggle entityType={EIssuesStoreType.PROFILE} entityId={userId} />}
      <FiltersDropdown title={t("common.display")} placement="bottom-end">
        <DisplayFiltersSelection
          layoutDisplayFiltersOptions={
            activeLayout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.profile_issues.layoutOptions[activeLayout] : undefined
          }
          displayFilters={issueFilters?.displayFilters ?? {}}
          handleDisplayFiltersUpdate={handleDisplayFilters}
          displayProperties={issueFilters?.displayProperties ?? {}}
          handleDisplayPropertiesUpdate={handleDisplayProperties}
        />
      </FiltersDropdown>
    </div>
  );
});
