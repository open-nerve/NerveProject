/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactElement } from "react";
import { createContext } from "react";
// lib
import { tokenManager } from "@/lib/auth/api-client";
// store
import { RootStore } from "@/store/root.store";

export let rootStore = new RootStore();

export const StoreContext = createContext<RootStore>(rootStore);

const initializeStore = () => {
  const newRootStore = rootStore ?? new RootStore();
  if (typeof window === "undefined") return newRootStore;
  if (!rootStore) rootStore = newRootStore;
  return newRootStore;
};

export const store = initializeStore();

// A tab that signs out, or follows another tab's sign-in as another account, starts again with new stores
// (M2 design 7.1): nothing of the account it showed stays on screen. A sign-in after a sign-out finds them
// new already.
let loginId = tokenManager.state.loginId;
tokenManager.subscribe(() => {
  const next = tokenManager.state.loginId;
  if (loginId !== undefined && next !== loginId) store.resetOnSignOut();
  loginId = next;
});

export function StoreProvider({ children }: { children: ReactElement }) {
  return <StoreContext.Provider value={store}>{children}</StoreContext.Provider>;
}
