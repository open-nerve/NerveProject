import type { IssueCreate } from "@nerve/api-client";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import React, { useRef } from "react";
import { observer } from "mobx-react";
// nerve constants
import { SPREADSHEET_PROPERTY_LIST } from "@nerve/constants";
// types
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
// local imports
import type { TRenderQuickActions } from "../list/list-view-types";
import { QuickAddIssueRoot, SpreadsheetAddIssueButton } from "../quick-add";
import { SpreadsheetTable } from "./spreadsheet-table";

type Props = {
  displayProperties: IIssueDisplayProperties;
  displayFilters: IIssueDisplayFilterOptions;
  handleDisplayFilterUpdate: (data: Partial<IIssueDisplayFilterOptions>) => void;
  issueIds: string[] | undefined;
  quickActions: TRenderQuickActions;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  canEditProperties: (projectId: string | undefined) => boolean;
  canLoadMoreIssues: boolean;
  loadMoreIssues: () => void;
  enableQuickCreateIssue?: boolean;
  disableIssueCreation?: boolean;
};

export const SpreadsheetView = observer(function SpreadsheetView(props: Props) {
  const {
    displayProperties,
    displayFilters,
    handleDisplayFilterUpdate,
    issueIds,
    quickActions,
    updateIssue,
    quickAddCallback,
    canEditProperties,
    enableQuickCreateIssue,
    disableIssueCreation,
    canLoadMoreIssues,
    loadMoreIssues,
  } = props;
  // refs
  const containerRef = useRef<HTMLDivElement | null>(null);
  const portalRef = useRef<HTMLDivElement | null>(null);
  const spreadsheetColumnsList = SPREADSHEET_PROPERTY_LIST;

  if (!issueIds || issueIds.length === 0) return <></>;
  return (
    <div className="relative flex h-full w-full flex-col overflow-x-hidden bg-layer-1 whitespace-nowrap text-secondary">
      <div ref={portalRef} className="spreadsheet-menu-portal" />
      <div ref={containerRef} className="vertical-scrollbar horizontal-scrollbar scrollbar-lg h-full w-full">
        <SpreadsheetTable
          displayProperties={displayProperties}
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
          issueIds={issueIds}
          portalElement={portalRef}
          quickActions={quickActions}
          updateIssue={updateIssue}
          canEditProperties={canEditProperties}
          containerRef={containerRef}
          canLoadMoreIssues={canLoadMoreIssues}
          loadMoreIssues={loadMoreIssues}
          spreadsheetColumnsList={spreadsheetColumnsList}
        />
      </div>
      <div className="border-t border-subtle">
        <div className="sticky bottom-0 left-0 z-5">
          {enableQuickCreateIssue && !disableIssueCreation && (
            <QuickAddIssueRoot
              layout={EIssueLayoutTypes.SPREADSHEET}
              QuickAddButton={SpreadsheetAddIssueButton}
              quickAddCallback={quickAddCallback}
            />
          )}
        </div>
      </div>
    </div>
  );
});
