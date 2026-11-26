/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentProps } from "react";
// nerve imports
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
import { EIssueFilterType, ISSUE_STORE_TO_FILTERS_MAP } from "@nerve/constants";
import type { EIssueLayoutTypes, EIssuesStoreType, TSupportedFilterForUpdate } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// local imports
import type { DisplayFiltersSelection } from "./issue-layouts/filters";

/**
 * A list page whose header's controls show and change its own preferences: the project's, a cycle's or a module's; or
 * its draft of its view, a project's view's (P6a spec 2.5).
 */
export type PageDisplayStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.PROJECT_VIEW;

/** What a list page's header's controls show of the page's preferences, and their changes. */
type PageDisplay = {
  /** The page's layout, as its preferences hold it. */
  layout: EIssueLayoutTypes | undefined;
  changeLayout: (layout: EIssueLayoutTypes) => void;
  /** The display panel's: the page's own options of its layout, the page's display and their changes. */
  display: ComponentProps<typeof DisplayFiltersSelection>;
};

/**
 * The preferences a list page's header shows and changes, at any width (P5b spec 2.6; P7c spec 2.7, R8), by the page's
 * store type: the page's own, read by the page's id as its address gives it (never by the router store's project,
 * which lags a commit when the address changes project, re-77 M2), changed by the page's update, which follows the
 * change as the page's commands follow it (useIssuesActions), with the page's own display options (a cycle's never
 * group by cycle, P5b R9; a module's never by module, P6b R12).
 */
export function usePageDisplay(storeType: PageDisplayStoreType, entityId: string): PageDisplay {
  const issueFilters = useIssues(storeType).issuesFilter.getIssueFilters(entityId);
  const { updateFilters } = useIssuesActions(storeType);
  const layout = issueFilters?.displayFilters?.layout;
  const update = (filterType: TSupportedFilterTypeForUpdate, filters: TSupportedFilterForUpdate) =>
    void updateFilters(entityId, filterType, filters);

  return {
    layout,
    changeLayout: (next) => update(EIssueFilterType.DISPLAY_FILTERS, { layout: next }),
    display: {
      layoutDisplayFiltersOptions: layout ? ISSUE_STORE_TO_FILTERS_MAP[storeType]?.layoutOptions[layout] : undefined,
      displayFilters: issueFilters?.displayFilters ?? {},
      handleDisplayFiltersUpdate: (change) => update(EIssueFilterType.DISPLAY_FILTERS, change),
      displayProperties: issueFilters?.displayProperties ?? {},
      handleDisplayPropertiesUpdate: (change) => update(EIssueFilterType.DISPLAY_PROPERTIES, change),
    },
  };
}
