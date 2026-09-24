/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { lazy } from "react";
// nerve imports
import type { TProfileSettingsTabs } from "@nerve/types";

export const PROFILE_SETTINGS_PAGES_MAP: Record<TProfileSettingsTabs, React.LazyExoticComponent<React.FC>> = {
  general: lazy(() => import("./general").then((m) => ({ default: m.GeneralProfileSettings }))),
  preferences: lazy(() => import("./preferences").then((m) => ({ default: m.PreferencesProfileSettings }))),
  security: lazy(() => import("./security").then((m) => ({ default: m.SecurityProfileSettings }))),
  "api-tokens": lazy(() => import("./api-tokens").then((m) => ({ default: m.APITokensProfileSettings }))),
};
