/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// ui
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
// hooks
import { SpreadsheetHeaderColumn } from "./spreadsheet-header-column";

interface Props {
  displayProperties: IIssueDisplayProperties;
  displayFilters: IIssueDisplayFilterOptions;
  handleDisplayFilterUpdate: (data: Partial<IIssueDisplayFilterOptions>) => void;
  spreadsheetColumnsList: (keyof IIssueDisplayProperties)[];
}

export const SpreadsheetHeader = observer(function SpreadsheetHeader(props: Props) {
  const { displayProperties, displayFilters, handleDisplayFilterUpdate, spreadsheetColumnsList } = props;

  return (
    <thead className="sticky top-0 left-0 z-[12] border-b-[0.5px] border-subtle">
      <tr>
        {/* Single header column containing both identifier and workitem */}
        <th
          className="group/list-header left-0 z-[15] h-11 min-w-60 border-r-[0.5px] border-subtle bg-layer-1 text-13 font-medium md:sticky"
          tabIndex={-1}
        >
          <div className="flex h-full w-full items-center gap-2 px-page-x">
            {/* Workitem header section */}
            <div className="flex h-full min-w-80 flex-grow items-center gap-1 py-2.5">
              <span className="text-13 font-medium">Work items</span>
            </div>
          </div>
        </th>

        {spreadsheetColumnsList.map((property) => (
          <SpreadsheetHeaderColumn
            key={property}
            property={property}
            displayProperties={displayProperties}
            displayFilters={displayFilters}
            handleDisplayFilterUpdate={handleDisplayFilterUpdate}
          />
        ))}
      </tr>
    </thead>
  );
});
