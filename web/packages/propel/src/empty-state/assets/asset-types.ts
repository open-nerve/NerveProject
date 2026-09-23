/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Horizontal Stack Asset Types
export type HorizontalStackAssetType =
  | "customer"
  | "intake"
  | "label"
  | "members"
  | "project"
  | "settings"
  | "state"
  | "token"
  | "unknown"
  | "update"
  | "webhook"
  | "work-item";

// Vertical Stack Asset Types
export type VerticalStackAssetType =
  | "archived-cycle"
  | "archived-module"
  | "archived-work-item"
  | "customer"
  | "cycle"
  | "dashboard"
  | "draft"
  | "error-404"
  | "invalid-link"
  | "module"
  | "no-access"
  | "project"
  | "server-error"
  | "view"
  | "work-item";

// Illustration Asset Types
export type IllustrationAssetType = "inbox" | "search";

// Combined Asset Types for Compact (uses horizontal + illustration)
export type CompactAssetType = HorizontalStackAssetType | IllustrationAssetType;

// Combined Asset Types for Detailed (uses vertical + illustration)
export type DetailedAssetType = VerticalStackAssetType | IllustrationAssetType;
