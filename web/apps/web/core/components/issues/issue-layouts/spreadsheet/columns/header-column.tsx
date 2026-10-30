/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

//ui
import { Eraser } from "lucide-react";
import {
  ArrowNarrowRightOutline,
  ChevronDownOutline,
  SortAscendingOutline,
  SortDescendingOutline,
  TickOutline,
} from "@makeplane/propel/icons";
// constants
// i18n
import { useTranslation } from "@nerve/i18n";
// types
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties, TIssueOrderByOptions } from "@nerve/types";
import { CustomMenu, Row } from "@nerve/ui";
import { spreadsheetSort } from "../spreadsheet-sort";
import { SpreadSheetPropertyIcon } from "../../utils";

interface Props {
  property: keyof IIssueDisplayProperties;
  displayFilters: IIssueDisplayFilterOptions;
  handleDisplayFilterUpdate: (data: Partial<IIssueDisplayFilterOptions>) => void;
  onClose: () => void;
}

export function HeaderColumn(props: Props) {
  const { displayFilters, handleDisplayFilterUpdate, property, onClose } = props;
  // i18n
  const { t } = useTranslation();
  const { propertyDetails, selectedOrder, direction } = spreadsheetSort(property, displayFilters);
  const handleOrderBy = (order: TIssueOrderByOptions) => handleDisplayFilterUpdate({ order_by: order });

  if (!propertyDetails) return null;

  return (
    <CustomMenu
      customButtonClassName="clickable !w-full"
      customButtonTabIndex={-1}
      className="!w-full"
      customButton={
        <Row className="flex w-full cursor-pointer items-center justify-between gap-1.5 py-2 text-13 text-secondary hover:text-primary">
          <div className="flex items-center gap-1.5">
            <SpreadSheetPropertyIcon iconKey={propertyDetails.icon} className="h-4 w-4 text-placeholder" />
            {t(propertyDetails.i18n_title)}
          </div>
          <div className="ml-3 flex">
            {direction !== "none" && (
              <div className="flex h-3.5 w-3.5 items-center justify-center rounded-full">
                {direction === "descending" ? (
                  <SortDescendingOutline className="h-3 w-3" />
                ) : (
                  <SortAscendingOutline className="h-3 w-3" />
                )}
              </div>
            )}
            <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />
          </div>
        </Row>
      }
      onMenuClose={onClose}
      placement="bottom-start"
      closeOnSelect
    >
      <CustomMenu.MenuItem onClick={() => handleOrderBy(propertyDetails.ascendingOrderKey)}>
        <div
          className={`flex items-center justify-between gap-1.5 px-1 ${
            selectedOrder === propertyDetails.ascendingOrderKey ? "text-primary" : "text-secondary hover:text-primary"
          }`}
        >
          <div className="flex items-center gap-2">
            <SortDescendingOutline className="h-3 w-3 stroke-[1.5]" />
            <span>{propertyDetails.ascendingOrderTitle}</span>
            <ArrowNarrowRightOutline className="h-3 w-3" />
            <span>{propertyDetails.descendingOrderTitle}</span>
          </div>

          {selectedOrder === propertyDetails.ascendingOrderKey && <TickOutline className="h-3 w-3" />}
        </div>
      </CustomMenu.MenuItem>
      <CustomMenu.MenuItem onClick={() => handleOrderBy(propertyDetails.descendingOrderKey)}>
        <div
          className={`flex items-center justify-between gap-1.5 px-1 ${
            selectedOrder === propertyDetails.descendingOrderKey ? "text-primary" : "text-secondary hover:text-primary"
          }`}
        >
          <div className="flex items-center gap-2">
            <SortAscendingOutline className="h-3 w-3 stroke-[1.5]" />
            <span>{propertyDetails.descendingOrderTitle}</span>
            <ArrowNarrowRightOutline className="h-3 w-3" />
            <span>{propertyDetails.ascendingOrderTitle}</span>
          </div>

          {selectedOrder === propertyDetails.descendingOrderKey && <TickOutline className="h-3 w-3" />}
        </div>
      </CustomMenu.MenuItem>
      {direction !== "none" && selectedOrder !== "-created_at" && (
        <CustomMenu.MenuItem className="mt-0.5" onClick={() => handleOrderBy("-created_at")}>
          <div className="flex items-center gap-2 px-1">
            <Eraser className="h-3 w-3" />
            <span>{t("common.actions.clear_sorting")}</span>
          </div>
        </CustomMenu.MenuItem>
      )}
    </CustomMenu>
  );
}
