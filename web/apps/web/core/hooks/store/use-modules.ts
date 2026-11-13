/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IModuleStore } from "@/store/planning/module.store";

/** The modules of the session's stores (RootStore.planning.modules). */
export const useModules = (): IModuleStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useModules must be used within StoreProvider");
  return context.planning.modules;
};
