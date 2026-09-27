/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createClient } from "@nerve/api-client";
import type { ApiClient } from "@nerve/api-client";
import { authMiddleware } from "./auth-middleware";
import { leaseLock, webLock } from "./refresh-lock";
import type { RefreshLock } from "./refresh-lock";
import { AUTH_KEY, TokenManager } from "./token-manager";

// The web app's clients of nerve's API and its token manager (M2 design 7.1). The clients send to the
// page's own origin, where nerve serves both the app and the API.

/** The client of the operations that need no token: the instance, sign-in and sign-up, and the token manager's own. */
export const publicClient = createClient();

/** n random bytes as hexadecimal, from crypto.getRandomValues, which pages on plain HTTP have too. */
function randomHex(bytes: number): string {
  return Array.from(crypto.getRandomValues(new Uint8Array(bytes)), (b) => b.toString(16).padStart(2, "0")).join("");
}

/** Calls listener with the key of every change another tab makes to localStorage (a null key: it was cleared). */
function onStorage(listener: (key: string | null) => void): () => void {
  const handle = (event: StorageEvent) => {
    if (event.storageArea === localStorage) listener(event.key);
  };
  window.addEventListener("storage", handle);
  return () => window.removeEventListener("storage", handle);
}

/**
 * navigator.locks where the page has it (HTTPS, localhost); else, as on plain HTTP at a LAN address, the
 * lease in localStorage.
 */
function refreshLock(): RefreshLock {
  if ("locks" in navigator) return webLock(navigator.locks);
  return leaseLock({ storage: localStorage, onStorage, now: Date.now, tabId: randomHex(16) });
}

export const tokenManager = new TokenManager({
  storage: localStorage,
  lock: refreshLock(),
  client: publicClient,
  now: Date.now,
  randomHex,
});

// Another tab signed in, signed out or refreshed: the token manager reads the record there now.
onStorage((key) => {
  if (key === AUTH_KEY || key === null) tokenManager.handleStorageChange();
});

// The session is decided once, as the app loads: before the stores exist, so they start with it.
void tokenManager.start();

/**
 * The client of every other operation for the stores of the session loginId (undefined: none): the access
 * token on each request, a 401 renewed once, and nothing sent once the tab is in another session.
 */
export function apiFor(loginId: string | undefined): ApiClient {
  const api = createClient();
  api.use(authMiddleware(tokenManager, loginId));
  return api;
}
