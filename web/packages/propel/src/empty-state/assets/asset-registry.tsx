/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import type {
  CompactAssetType,
  DetailedAssetType,
  HorizontalStackAssetType,
  IllustrationAssetType,
  VerticalStackAssetType,
} from "./asset-types";
import {
  IntakeHorizontalStackIllustration,
  LabelHorizontalStackIllustration,
  MembersHorizontalStackIllustration,
  ProjectHorizontalStackIllustration,
  SettingsHorizontalStackIllustration,
  StateHorizontalStackIllustration,
  TokenHorizontalStackIllustration,
  UnknownHorizontalStackIllustration,
  WebhookHorizontalStackIllustration,
  WorkItemHorizontalStackIllustration,
} from "./horizontal-stack";
import { InboxIllustration, SearchIllustration } from "./illustration";
import {
  ArchivedCycleVerticalStackIllustration,
  ArchivedModuleVerticalStackIllustration,
  ArchivedWorkItemVerticalStackIllustration,
  CycleVerticalStackIllustration,
  DraftVerticalStackIllustration,
  Error404VerticalStackIllustration,
  InvalidLinkVerticalStackIllustration,
  ModuleVerticalStackIllustration,
  NoAccessVerticalStackIllustration,
  ProjectVerticalStackIllustration,
  ServerErrorVerticalStackIllustration,
  ViewVerticalStackIllustration,
  WorkItemVerticalStackIllustration,
} from "./vertical-stack";

// Horizontal Stack Asset Registry
const HORIZONTAL_STACK_ASSETS: Record<HorizontalStackAssetType, React.ComponentType<{ className?: string }>> = {
  intake: IntakeHorizontalStackIllustration,
  label: LabelHorizontalStackIllustration,
  members: MembersHorizontalStackIllustration,
  project: ProjectHorizontalStackIllustration,
  settings: SettingsHorizontalStackIllustration,
  state: StateHorizontalStackIllustration,
  token: TokenHorizontalStackIllustration,
  unknown: UnknownHorizontalStackIllustration,
  webhook: WebhookHorizontalStackIllustration,
  "work-item": WorkItemHorizontalStackIllustration,
};

// Vertical Stack Asset Registry
const VERTICAL_STACK_ASSETS: Record<VerticalStackAssetType, React.ComponentType<{ className?: string }>> = {
  "archived-cycle": ArchivedCycleVerticalStackIllustration,
  "archived-module": ArchivedModuleVerticalStackIllustration,
  "archived-work-item": ArchivedWorkItemVerticalStackIllustration,
  cycle: CycleVerticalStackIllustration,
  draft: DraftVerticalStackIllustration,
  "error-404": Error404VerticalStackIllustration,
  "invalid-link": InvalidLinkVerticalStackIllustration,
  module: ModuleVerticalStackIllustration,
  "no-access": NoAccessVerticalStackIllustration,
  project: ProjectVerticalStackIllustration,
  "server-error": ServerErrorVerticalStackIllustration,
  view: ViewVerticalStackIllustration,
  "work-item": WorkItemVerticalStackIllustration,
};

// Illustration Asset Registry
const ILLUSTRATION_ASSETS: Record<IllustrationAssetType, React.ComponentType<{ className?: string }>> = {
  inbox: InboxIllustration,
  search: SearchIllustration,
};

// Helper functions to get assets
export const getCompactAsset = (assetKey: CompactAssetType, className?: string): React.ReactNode => {
  const AssetComponent =
    (HORIZONTAL_STACK_ASSETS[assetKey as HorizontalStackAssetType] as React.ComponentType<{ className?: string }>) ||
    ILLUSTRATION_ASSETS[assetKey as IllustrationAssetType];

  if (!AssetComponent) {
    console.warn(`Asset "${assetKey}" not found in compact asset registry`);
    return null;
  }

  return <AssetComponent className={className} />;
};

export const getDetailedAsset = (assetKey: DetailedAssetType, className?: string): React.ReactNode => {
  const AssetComponent =
    (VERTICAL_STACK_ASSETS[assetKey as VerticalStackAssetType] as React.ComponentType<{ className?: string }>) ||
    ILLUSTRATION_ASSETS[assetKey as IllustrationAssetType];

  if (!AssetComponent) {
    console.warn(`Asset "${assetKey}" not found in detailed asset registry`);
    return null;
  }

  return <AssetComponent className={className} />;
};
