/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { SPREADSHEET_PROPERTY_DETAILS } from "@nerve/constants";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { projectListQuery } from "@/lib/issue-filters";

/**
 * A column's sort, which both the menu and the accessible column state project from the confirmed preference; none for
 * a column that sorts by nothing (a work item's modules and cycle until P7b, P7a spec R21), whose header is its title.
 */
export function spreadsheetSort(property: keyof IIssueDisplayProperties, display: IIssueDisplayFilterOptions) {
  const propertyDetails = SPREADSHEET_PROPERTY_DETAILS[property];
  if (propertyDetails?.ascendingOrderKey === undefined) return undefined;
  const selectedOrder = projectListQuery({}, display).order_by;
  const active =
    selectedOrder === propertyDetails.ascendingOrderKey || selectedOrder === propertyDetails.descendingOrderKey;
  const direction = !active ? "none" : selectedOrder?.startsWith("-") ? "descending" : "ascending";
  return { propertyDetails, selectedOrder, direction } as const;
}
