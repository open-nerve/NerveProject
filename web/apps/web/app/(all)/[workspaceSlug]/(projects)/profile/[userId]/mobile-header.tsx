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
import { EIssueFilterType, ISSUE_LAYOUTS, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
// nerve i18n
import { useTranslation } from "@nerve/i18n";
// icons
import { ChevronDownOutline } from "@makeplane/propel/icons";
// types
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@nerve/types";
// ui
import { CustomMenu } from "@nerve/ui";
// components
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { IssueLayoutIcon } from "@/components/issues/issue-layouts/layout-icon";
// hooks
import { useIssues } from "@/hooks/store/use-issues";

export const ProfileIssuesMobileHeader = observer(function ProfileIssuesMobileHeader() {
  // nerve i18n
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
    <div className="flex justify-evenly border-b border-subtle py-2 md:hidden">
      <CustomMenu
        maxHeight="md"
        className="flex flex-grow justify-center text-13 text-secondary"
        placement="bottom-start"
        customButton={
          <div className="flex-center flex text-13 text-secondary">
            {t("common.layout")}
            <ChevronDownOutline className="my-auto ml-2 h-4 w-4 text-secondary" />
          </div>
        }
        customButtonClassName="flex flex-center text-secondary text-13"
        closeOnSelect
      >
        {ISSUE_LAYOUTS.map((layout) => {
          if (layout.key === "spreadsheet" || layout.key === "calendar") return;
          return (
            <CustomMenu.MenuItem
              key={layout.key}
              onClick={() => {
                void handleLayoutChange(layout.key === "list" ? EIssueLayoutTypes.LIST : EIssueLayoutTypes.KANBAN);
              }}
              className="flex items-center gap-2"
            >
              <IssueLayoutIcon layout={layout.key} className="h-3 w-3" />
              <div className="text-tertiary">{t(layout.i18n_title)}</div>
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
      <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
        <FiltersDropdown
          title={t("common.display")}
          placement="bottom-end"
          menuButton={
            <div className="flex-center flex text-13 text-secondary">
              {t("common.display")}
              <ChevronDownOutline className="ml-2 h-4 w-4 text-secondary" />
            </div>
          }
        >
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
    </div>
  );
});
