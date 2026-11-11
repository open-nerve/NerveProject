/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { ICycleStore } from "@/store/planning/cycle.store";

/** The cycles of the session's stores (RootStore.planning.cycles). */
export const useCycles = (): ICycleStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useCycles must be used within StoreProvider");
  return context.planning.cycles;
};
