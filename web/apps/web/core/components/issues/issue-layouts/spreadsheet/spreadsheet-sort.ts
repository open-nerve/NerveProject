/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { SPREADSHEET_PROPERTY_DETAILS } from "@nerve/constants";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { projectListQuery } from "@/lib/issue-filters";

/** Both the menu and the accessible column state project the confirmed preference. */
export function spreadsheetSort(property: keyof IIssueDisplayProperties, display: IIssueDisplayFilterOptions) {
  const propertyDetails = SPREADSHEET_PROPERTY_DETAILS[property];
  const selectedOrder = projectListQuery({}, display).order_by;
  const active =
    propertyDetails &&
    (selectedOrder === propertyDetails.ascendingOrderKey || selectedOrder === propertyDetails.descendingOrderKey);
  const direction = !active ? "none" : selectedOrder?.startsWith("-") ? "descending" : "ascending";
  return { propertyDetails, selectedOrder, direction } as const;
}
