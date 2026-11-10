/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { ICycleFilterStore } from "@/store/planning/cycle_filter.store";

/** What each project's cycles list and archive are filtered and searched by (RootStore.planning.cycleFilters). */
export const useCycleFilters = (): ICycleFilterStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useCycleFilters must be used within StoreProvider");
  return context.planning.cycleFilters;
};
