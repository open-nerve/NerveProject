/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IIntakeStore } from "@/store/issue/intake/intake.store";

/** The intake of the session's stores (the issue root's, RootStore.issue.intake; P5a spec 2.3). */
export const useIntake = (): IIntakeStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useIntake must be used within StoreProvider");
  return context.issue.intake;
};
