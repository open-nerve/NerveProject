/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IModuleFilterStore } from "@/store/planning/module_filter.store";

/** What each project's modules list and archive are filtered and searched by (RootStore.planning.moduleFilters). */
export const useModuleFilters = (): IModuleFilterStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useModuleFilters must be used within StoreProvider");
  return context.planning.moduleFilters;
};
