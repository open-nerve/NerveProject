/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IProjectPreferencesStore } from "@/store/project/preferences.store";

export const useProjectPreferences = (): IProjectPreferencesStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useProjectPreferences must be used within StoreProvider");
  return context.projectRoot.preferences;
};
