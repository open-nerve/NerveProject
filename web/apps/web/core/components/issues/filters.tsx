/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { PreferencesOutline } from "@makeplane/propel/icons";
// nerve imports
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssueFilterType, ISSUE_STORE_TO_FILTERS_MAP } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { TSupportedFilterForUpdate } from "@nerve/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@nerve/types";
// hooks
import { usePageIssueFilters } from "@/hooks/store/use-page-issue-filters";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// local imports
import { WorkItemFiltersToggle } from "../work-item-filters/filters-toggle";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  LayoutSelection,
  MobileLayoutSelection,
} from "./issue-layouts/filters";

type Props = {
  /** The list page whose own preferences these controls show and change: the project's, or a cycle's. */
  storeType: EIssuesStoreType.PROJECT | EIssuesStoreType.CYCLE;
  /** The page's id, as the route gives it: the project's, or the cycle's. */
  entityId: string;
};
const LAYOUTS = [
  EIssueLayoutTypes.LIST,
  EIssueLayoutTypes.KANBAN,
  EIssueLayoutTypes.CALENDAR,
  EIssueLayoutTypes.SPREADSHEET,
];

/**
 * The layouts, the toggle of the filters and the display of a list page (P5b spec 2.6), by its store type: the page's
 * own preferences, their change followed as the page's commands follow it, and the page's own display options (a
 * cycle's never group by cycle, R9).
 */
export const HeaderFilters = observer(function HeaderFilters(props: Props) {
  const { storeType, entityId } = props;
  // i18n
  const { t } = useTranslation();
  // store hooks
  const issueFilters = usePageIssueFilters(storeType);
  const { updateFilters } = useIssuesActions(storeType);
  // derived values
  const activeLayout = issueFilters?.displayFilters?.layout;
  const layoutDisplayFiltersOptions = activeLayout
    ? ISSUE_STORE_TO_FILTERS_MAP[storeType]?.layoutOptions[activeLayout]
    : undefined;

  const update = (filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) =>
    void updateFilters(entityId, filterType, filters);
  const handleLayoutChange = (layout: EIssueLayoutTypes) => update(EIssueFilterType.DISPLAY_FILTERS, { layout });

  return (
    <>
      <div className="hidden @4xl:flex">
        <LayoutSelection layouts={LAYOUTS} onChange={handleLayoutChange} selectedLayout={activeLayout} />
      </div>
      <div className="flex @4xl:hidden">
        <MobileLayoutSelection layouts={LAYOUTS} onChange={handleLayoutChange} activeLayout={activeLayout} />
      </div>
      <WorkItemFiltersToggle entityType={storeType} entityId={entityId} />
      <FiltersDropdown
        miniIcon={<PreferencesOutline className="size-3.5" />}
        title={t("common.display")}
        placement="bottom-end"
      >
        <DisplayFiltersSelection
          layoutDisplayFiltersOptions={layoutDisplayFiltersOptions}
          displayFilters={issueFilters?.displayFilters ?? {}}
          handleDisplayFiltersUpdate={(change) => update(EIssueFilterType.DISPLAY_FILTERS, change)}
          displayProperties={issueFilters?.displayProperties ?? {}}
          handleDisplayPropertiesUpdate={(change) => update(EIssueFilterType.DISPLAY_PROPERTIES, change)}
        />
      </FiltersDropdown>
    </>
  );
});
