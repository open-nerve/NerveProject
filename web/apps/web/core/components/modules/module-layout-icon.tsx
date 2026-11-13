/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { GridOutline, ListOutline } from "@makeplane/propel/icons";
// nerve imports
import type { ModuleLayout } from "@nerve/constants";

/** The icon of each layout of a modules list: its rows, its cards. */
const ICONS = { list: ListOutline, board: GridOutline } as const satisfies Record<ModuleLayout, unknown>;

/** The icon of a layout of a project's modules list (P6a spec 2.4), beside the layout's name, which says it. */
export function ModuleLayoutIcon({ layout }: { layout: ModuleLayout }) {
  const Icon = ICONS[layout];
  return <Icon width={14} height={14} className="flex-shrink-0" aria-hidden="true" />;
}
