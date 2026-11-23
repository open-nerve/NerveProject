/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

const PROJECT_VIEW_TAB_INDICES = ["name", "description", "filters", "cancel", "submit"];

export enum ETabIndices {
  PROJECT_VIEW = "project-view",
}

export const TAB_INDEX_MAP: Record<ETabIndices, string[]> = {
  [ETabIndices.PROJECT_VIEW]: PROJECT_VIEW_TAB_INDICES,
};
