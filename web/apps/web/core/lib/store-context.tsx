/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactElement } from "react";
import { createContext, useSyncExternalStore } from "react";
// nerve imports
import { FALLBACK_LANGUAGE, setLanguage } from "@nerve/i18n";
// lib
import { apiFor, tokenManager } from "@/lib/auth/api-client";
// store
import { RootStore } from "@/store/root.store";

// Each session of the tab has its own RootStore, whose client is bound to that session (M2 design 7.1). The
// first is built for the session the token manager took up as the app loaded (api-client.ts starts it before
// the stores exist).
let loginId = tokenManager.state.loginId;

/**
 * The RootStore of the tab's session now: a new session replaces it, and this binding with it. Code outside
 * the components that has no RootStore of its own reads it here each time, never keeps it.
 */
export let rootStore = new RootStore(apiFor(loginId));

const listeners = new Set<() => void>();

/** Calls listener each time a new session starts, its RootStore already in place; returns the unsubscribe. */
export function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** The RootStore of the tab's session now, for StoreProvider. */
export const snapshot = (): RootStore => rootStore;

/**
 * A new session starts: the tab signed out or in, or followed another tab's sign-in, maybe as another account.
 * It gets a new RootStore, and the app renders again with it: nothing of the account shown before stays on
 * screen. The stores left behind keep their own RootStore, whose account's stores and client are the
 * old session's: nothing they still do reaches the new one. The theme and the language were the account's; the
 * new session shows the defaults until its profile sets them.
 */
function startSession(next: string | undefined): void {
  loginId = next;
  localStorage.setItem("theme", "system");
  void setLanguage(FALLBACK_LANGUAGE);
  rootStore = new RootStore(apiFor(next), rootStore);
  for (const listener of listeners) listener();
}

// Subscribed as this module loads, before any component subscribes to the session: by the time a component
// renders the new session, its RootStore is in place. A change of state within the session (its first refresh,
// a passing failure, a retry) keeps the RootStore.
tokenManager.subscribe(() => {
  if (tokenManager.state.loginId !== loginId) startSession(tokenManager.state.loginId);
});

export const StoreContext = createContext<RootStore>(rootStore);

/** Provides the RootStore of the tab's session now: every component below renders again when a new one starts. */
export function StoreProvider({ children }: { children: ReactElement }) {
  const store = useSyncExternalStore(subscribe, snapshot);
  return <StoreContext.Provider value={store}>{children}</StoreContext.Provider>;
}
