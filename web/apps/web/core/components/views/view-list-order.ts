/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { orderBy } from "lodash-es";
// nerve imports
import type { View } from "@nerve/api-client";
import type { TViewFiltersSortBy, TViewFiltersSortKey } from "@nerve/types";
import { getDate, satisfiesDateFilter } from "@nerve/utils";

/** The filters of a project's views list (P6a spec 2.3): its views' owners, and their dates of creation. */
export type ViewListFilters = {
  /** The owners whose views the list shows (owned_by_id); none: everyone's. */
  owned_by: string[];
  /** The dates of creation, each a date filter (satisfiesDateFilter) that a view's creation passes; none: any. */
  created_at: string[];
};

/** What the person chose of a project's views list: its search, its filters and its order. */
export type ViewListChoices = {
  /** The text the names of the views shown hold, whatever its case. */
  search: string;
  filters: ViewListFilters;
  /** What the list is ordered by, and which way. */
  sortKey: TViewFiltersSortKey;
  sortBy: TViewFiltersSortBy;
};

/** No filter of the list. */
export const NO_VIEW_FILTERS: ViewListFilters = { owned_by: [], created_at: [] };

/** A project's views list as its page opens: no search, no filter, the last changed first. */
export const FIRST_CHOICES: ViewListChoices = {
  search: "",
  filters: NO_VIEW_FILTERS,
  sortKey: "updated_at",
  sortBy: "desc",
};

/** What the list orders a view by, for each order. */
const ORDER_OF: Record<TViewFiltersSortKey, (view: View) => string | number> = {
  name: (view) => view.name.toLowerCase(),
  created_at: (view) => Date.parse(view.created_at),
  updated_at: (view) => Date.parse(view.updated_at),
};

/** Whether the view passes the filters: owned by one of their owners, created as each of their dates says. */
function passes(view: View, filters: ViewListFilters): boolean {
  const created = getDate(view.created_at);
  return (
    (filters.owned_by.length === 0 || filters.owned_by.includes(view.owned_by_id)) &&
    filters.created_at.every((date) => created !== undefined && satisfiesDateFilter(created, date))
  );
}

/**
 * The project's views the list shows (P6a spec 2.3): those whose names hold the search, whatever its case, its spaces
 * around aside, and that pass the filters, in the order chosen; views the order does not tell apart keep nerve's order
 * (the newest first).
 */
export function viewsShown(views: View[], choices: ViewListChoices): View[] {
  const searched = choices.search.trim().toLowerCase();
  const shown = views.filter((view) => view.name.toLowerCase().includes(searched) && passes(view, choices.filters));
  return orderBy(shown, [ORDER_OF[choices.sortKey]], [choices.sortBy]);
}
