/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import { MODULE_ORDER_BY_OPTIONS, MODULE_VIEW_LAYOUTS } from "@nerve/constants";
import type { ModuleLayout, ModuleOrderBy } from "@nerve/constants";
import { useLocalStorage } from "@nerve/hooks";

/** How a project's modules list and archive are laid out and ordered, and how the person chooses otherwise. */
type ModuleDisplay = {
  layout: ModuleLayout;
  orderBy: ModuleOrderBy;
  setLayout: (layout: ModuleLayout) => void;
  setOrderBy: (orderBy: ModuleOrderBy) => void;
};

const DEFAULT_LAYOUT: ModuleLayout = "list";
const DEFAULT_ORDER: ModuleOrderBy = "name";
/** The layouts of the list, and its orders: each option, and each but the manual one behind a "-" too. */
const LAYOUTS = new Set<unknown>(MODULE_VIEW_LAYOUTS.map(({ key }) => key));
const ORDERS = new Set<unknown>(
  MODULE_ORDER_BY_OPTIONS.flatMap(({ key }) => (key === "sort_order" ? [key] : [key, `-${key}`]))
);
const isLayout = (value: unknown): value is ModuleLayout => LAYOUTS.has(value);
const isOrderBy = (value: unknown): value is ModuleOrderBy => ORDERS.has(value);

/**
 * How a project's modules list and archive are laid out (rows or cards) and ordered (P6a spec 2.4, R4), as the person
 * chose them for the project: the browser keeps them (useLocalStorage), by the project's id, as the old pages did and as
 * the module's sidebar is kept (module_sidebar_collapsed): every reader of the project's (the list's header, a small
 * screen's layout menu, the list, its archive) gives and takes the same, and the page mounted again (a reload) reads
 * them back. They hold nothing of the account's, unlike the filters and the search, which stay in the session
 * (ModuleFilterStore). Rows ordered by name until the person chooses otherwise; a value the browser holds that is none
 * of the list's (an old layout, a manual order with a direction) gives the default.
 */
export function useModuleDisplay(projectId: string): ModuleDisplay {
  const layout = useLocalStorage<ModuleLayout>(`module-list-layout-${projectId}`, DEFAULT_LAYOUT);
  const orderBy = useLocalStorage<ModuleOrderBy>(`module-list-order-${projectId}`, DEFAULT_ORDER);
  return {
    layout: isLayout(layout.storedValue) ? layout.storedValue : DEFAULT_LAYOUT,
    orderBy: isOrderBy(orderBy.storedValue) ? orderBy.storedValue : DEFAULT_ORDER,
    setLayout: layout.setValue,
    setOrderBy: orderBy.setValue,
  };
}
