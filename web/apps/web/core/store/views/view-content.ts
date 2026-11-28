/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ViewCreate } from "@nerve/api-client";

/**
 * What a new view keeps besides who it is (its name, description and icon), in nerve's words (ViewCreate): its filters,
 * its display filters and its display properties. The view's dialog never shows them (P6a spec 2.4, R28): its page
 * changes them, which Update view saves.
 */
export type ViewContent = Pick<ViewCreate, "filters" | "display_filters" | "display_properties">;

/** Every property a view shows, all on: a new view's (P6a spec 2.4, R28). */
const EVERY_PROPERTY: ViewContent["display_properties"] = {
  assignee: true,
  start_date: true,
  due_date: true,
  labels: true,
  key: true,
  priority: true,
  state: true,
  sub_issue_count: true,
  link: true,
  attachment_count: true,
  created_on: true,
  updated_on: true,
  cycle: true,
  modules: true,
};

/**
 * The display nerve gives a view made without one (ViewCreate, navigation.yaml), in the layout given: without grouping,
 * the newest first, sub-issues and empty groups shown, the calendar by month without weekends.
 */
const displayMadeWithout = (layout: "list" | "spreadsheet"): ViewContent["display_filters"] => ({
  layout,
  group_by: null,
  sub_group_by: null,
  order_by: "-created_at",
  sub_issue: true,
  show_empty_groups: true,
  calendar: { layout: "month", show_weekends: false },
});

/**
 * What "Add view" makes a project's view with (P6a spec 2.4, R28): no filters, and the display nerve gives a view made
 * without one, the list layout; and every property shown.
 */
export const NEW_VIEW_CONTENT: ViewContent = {
  display_filters: displayMadeWithout("list"),
  display_properties: EVERY_PROPERTY,
};

/**
 * What "Add view" makes a workspace's own view with (M7/P6b spec 2.11, R8): NEW_VIEW_CONTENT in the spreadsheet's
 * layout, the one its page draws; no filters, and every property shown.
 */
export const WORKSPACE_VIEW_CONTENT: ViewContent = {
  display_filters: displayMadeWithout("spreadsheet"),
  display_properties: EVERY_PROPERTY,
};
