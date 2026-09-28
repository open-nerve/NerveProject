/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
// types
import type { IApiTokenStore } from "@/store/user/api-token.store";

/** The personal access tokens of the account of the tab's session now. */
export const useApiTokens = (): IApiTokenStore => useContext(StoreContext).user.apiTokens;
