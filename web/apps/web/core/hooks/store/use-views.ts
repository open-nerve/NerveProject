/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IViewStore } from "@/store/views/view.store";

/** The views of the session's stores (RootStore.views, P6a spec 2.1). */
export const useViews = (): IViewStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useViews must be used within StoreProvider");
  return context.views;
};
