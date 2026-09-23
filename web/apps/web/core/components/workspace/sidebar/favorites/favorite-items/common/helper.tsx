/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FAVORITE_ITEM_LINKS } from "@plane/constants";
import type { IFavorite } from "@plane/types";

export const generateFavoriteItemLink = (workspaceSlug: string, favorite: IFavorite) => {
  const getLink = FAVORITE_ITEM_LINKS[favorite.entity_type];

  if (!getLink) {
    console.error(`Unrecognized favorite entity type: ${favorite.entity_type}`);
    return `/${workspaceSlug}`;
  }

  return `/${workspaceSlug}/projects/${favorite.project_id}/${getLink(favorite)}`;
};
