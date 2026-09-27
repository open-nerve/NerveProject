/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useSyncExternalStore } from "react";
import { tokenManager } from "./api-client";
import type { SessionState } from "./token-manager";

const snapshot = () => tokenManager.state;

/** The session as the token manager has it; the component renders again whenever it changes. */
export function useSession(): SessionState {
  return useSyncExternalStore(tokenManager.subscribe, snapshot);
}
