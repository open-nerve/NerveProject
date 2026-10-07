/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { Workspace } from "@nerve/api-client";

/** The workspaces by name, as the sidebar lists them: a new list, the one given unchanged. */
export const orderWorkspacesList = (workspaces: readonly Workspace[]): Workspace[] =>
  workspaces.toSorted((a, b) => a.name.localeCompare(b.name));
