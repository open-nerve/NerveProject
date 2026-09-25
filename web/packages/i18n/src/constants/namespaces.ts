/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const NAMESPACES = [
  "accessibility",
  "auth",
  "common",
  "cycle",
  "empty-state",
  "home",
  "inbox",
  "module",
  "navigation",
  "notification",
  "power-k",
  "project",
  "project-settings",
  "settings",
  "work-item",
  "workspace",
  "workspace-settings",
] as const;

type TNamespace = (typeof NAMESPACES)[number];

export const DEFAULT_NAMESPACE: TNamespace = "common";
