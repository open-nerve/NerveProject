/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { INotificationStore } from "@/store/notification/notification.store";

/** The caller's notifications of the session's stores (RootStore.notification). */
export const useNotifications = (): INotificationStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useNotifications must be used within StoreProvider");
  return context.notification;
};
