/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IFavorite, TFavoriteEntityType } from "@plane/types";

// The link of a favourite, under its project's address; a folder only groups favourites and has none
export const FAVORITE_ITEM_LINKS: Partial<Record<TFavoriteEntityType, (favorite: IFavorite) => string>> = {
  project: () => "issues",
  cycle: (favorite) => `cycles/${favorite.entity_identifier}`,
  module: (favorite) => `modules/${favorite.entity_identifier}`,
  view: (favorite) => `views/${favorite.entity_identifier}`,
};
