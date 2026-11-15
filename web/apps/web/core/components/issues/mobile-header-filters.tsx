/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ChevronDownOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EIssueLayoutTypes } from "@nerve/types";
// local imports
import { DisplayFiltersSelection, FiltersDropdown, MobileLayoutSelection } from "./issue-layouts/filters";
import type { PageDisplayStoreType } from "./use-page-display";
import { usePageDisplay } from "./use-page-display";

type Props = {
  /** The list page whose own preferences the row shows and changes: the project's, a cycle's or a module's. */
  storeType: PageDisplayStoreType;
  /** The page's id, as the route gives it: the project's, the cycle's or the module's. */
  entityId: string;
};
const LAYOUTS = [EIssueLayoutTypes.LIST, EIssueLayoutTypes.KANBAN, EIssueLayoutTypes.CALENDAR];

/**
 * The small screen's row of a list page's header (P7c spec 2.7, R8), by the page's store type: the page's layouts, the
 * list, the board and the calendar, whose trigger shows the page's, and the page's display, as the header's controls at
 * a wider width give them (HeaderFilters, by usePageDisplay); no filters. Whether the page shows the row is its caller's:
 * each page's wrapper gives it once nerve has given the page's preferences.
 */
export const MobileHeaderFilters = observer(function MobileHeaderFilters({ storeType, entityId }: Props) {
  // i18n
  const { t } = useTranslation();
  // the page's preferences
  const { layout, changeLayout, display } = usePageDisplay(storeType, entityId);

  return (
    <div className="z-[13] flex justify-evenly border-b border-subtle bg-surface-1 py-2 md:hidden">
      <MobileLayoutSelection layouts={LAYOUTS} onChange={changeLayout} activeLayout={layout} />
      <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
        <FiltersDropdown
          title={t("common.display")}
          placement="bottom-end"
          menuButton={
            <span className="flex items-center text-13 text-secondary">
              {t("common.display")}
              <ChevronDownOutline className="ml-2 h-4 w-4 text-secondary" />
            </span>
          }
        >
          <DisplayFiltersSelection {...display} />
        </FiltersDropdown>
      </div>
    </div>
  );
});
