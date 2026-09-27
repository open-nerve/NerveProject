/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactElement } from "react";
import { createContext } from "react";
// lib
import { apiFor, tokenManager } from "@/lib/auth/api-client";
// store
import { RootStore } from "@/store/root.store";

// The stores serve one session at a time, with a client bound to it (M2 design 7.1): first the session the
// token manager took up as the app loaded (api-client.ts starts it before the stores exist).
let loginId = tokenManager.state.loginId;

export let rootStore = new RootStore(apiFor(loginId));

export const StoreContext = createContext<RootStore>(rootStore);

const initializeStore = () => {
  const newRootStore = rootStore ?? new RootStore(apiFor(loginId));
  if (typeof window === "undefined") return newRootStore;
  if (!rootStore) rootStore = newRootStore;
  return newRootStore;
};

export const store = initializeStore();

// Whenever the tab's session changes, the stores start again for the new one: the tab signed out or in, or
// followed another tab's sign-in, maybe as another account. Nothing of the account it showed stays on screen,
// and nothing the old stores still do reaches the new session: their client is bound to the old one. A change
// of state within the session (its first refresh, a retry) keeps them.
tokenManager.subscribe(() => {
  const next = tokenManager.state.loginId;
  if (next !== loginId) store.resetOnSignOut(apiFor(next));
  loginId = next;
});

export function StoreProvider({ children }: { children: ReactElement }) {
  return <StoreContext.Provider value={store}>{children}</StoreContext.Provider>;
}
