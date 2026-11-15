/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { PreferencesOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EIssueLayoutTypes } from "@nerve/types";
// local imports
import { WorkItemFiltersToggle } from "../work-item-filters/filters-toggle";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  LayoutSelection,
  MobileLayoutSelection,
} from "./issue-layouts/filters";
import type { PageDisplayStoreType } from "./use-page-display";
import { usePageDisplay } from "./use-page-display";

type Props = {
  /** The list page whose own preferences these controls show and change: the project's, a cycle's or a module's. */
  storeType: PageDisplayStoreType;
  /** The page's id, as the route gives it: the project's, the cycle's or the module's. */
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
 * cycle's never group by cycle, P5b R9; a module's never by module, P6b R12), as the small screen's row gives its own
 * (usePageDisplay, MobileHeaderFilters).
 */
export const HeaderFilters = observer(function HeaderFilters(props: Props) {
  const { storeType, entityId } = props;
  // i18n
  const { t } = useTranslation();
  // the page's preferences
  const { layout, changeLayout, display } = usePageDisplay(storeType, entityId);

  return (
    <>
      <div className="hidden @4xl:flex">
        <LayoutSelection layouts={LAYOUTS} onChange={changeLayout} selectedLayout={layout} />
      </div>
      <div className="flex @4xl:hidden">
        <MobileLayoutSelection layouts={LAYOUTS} onChange={changeLayout} activeLayout={layout} />
      </div>
      <WorkItemFiltersToggle entityType={storeType} entityId={entityId} />
      <FiltersDropdown
        miniIcon={<PreferencesOutline className="size-3.5" />}
        title={t("common.display")}
        placement="bottom-end"
      >
        <DisplayFiltersSelection {...display} />
      </FiltersDropdown>
    </>
  );
});
