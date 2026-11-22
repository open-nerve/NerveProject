/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { NotificationTab, NotificationType } from "@nerve/api-client";
// services
import type { NotificationListQuery } from "@/services/notification.service";

/**
 * The filters of the caller's notifications, as the notifications page holds them (P4 spec 2.2, 2.5): a tab; in the
 * all tab, the senders to list (none, every one); the unread alone; the archived alone; the snoozed alone.
 */
export type NotificationFilters = {
  tab: NotificationTab;
  types: readonly NotificationType[];
  unread: boolean;
  archived: boolean;
  snoozed: boolean;
};

/** The toggles of the filters, in the order a key names those that are on. */
const TOGGLES = ["unread", "archived", "snoozed"] as const;

/** A page of the caller's notifications holds at most 100: nerve takes 1–100 (R21, the contract's T3). */
const PAGE_SIZE = 100;

/**
 * The senders the filters select, as nerve is asked for them: each once, in order (assigned, created, subscribed);
 * none in the mentions tab, where nerve takes no type (the contract's T2, R10).
 */
const typesOf = (filters: NotificationFilters): NotificationType[] =>
  filters.tab === "all" ? [...new Set(filters.types)].toSorted() : [];

/**
 * @description the filters as one string, the same for any filters that list the same notifications: the tab, the
 * types it selects (typesOf), and the toggles that are on, each part after a ";"; filtersOf gives back the filters it
 * names, so that what a list's key names is what it fetches (re-93 m5)
 */
export const filterKey = (filters: NotificationFilters): string =>
  [filters.tab, typesOf(filters).join(","), TOGGLES.filter((toggle) => filters[toggle]).join(",")].join(";");

/** @description the filters a key of filterKey names: the types in order, each once, none in the mentions tab */
export function filtersOf(key: string): NotificationFilters {
  const [tab, types, toggles] = key.split(";");
  const on = new Set(toggles.split(","));
  return {
    tab: tab as NotificationTab,
    types: types === "" ? [] : (types.split(",") as NotificationType[]),
    unread: on.has("unread"),
    archived: on.has("archived"),
    snoozed: on.has("snoozed"),
  };
}

/**
 * @description the query of a page of the notifications the filters select (P4 spec 2.2): always the tab, archived
 * and snoozed, false unless on (T7), and pages of 100 (T3); the types only in the all tab and only when there are some,
 * joined by commas (T2, T8: nerve refuses a type in the mentions tab and an empty one); read=false for the unread alone,
 * no read otherwise (T5: read=true lists the read alone); the cursor only as nerve answered it (T4)
 */
export function listQuery(filters: NotificationFilters, cursor?: string): NotificationListQuery {
  const types = typesOf(filters);
  return {
    tab: filters.tab,
    ...(types.length > 0 ? { type: types.join(",") } : {}),
    ...(filters.unread ? { read: "false" } : {}),
    archived: String(filters.archived),
    snoozed: String(filters.snoozed),
    limit: PAGE_SIZE,
    ...(cursor === undefined ? {} : { cursor }),
  };
}
