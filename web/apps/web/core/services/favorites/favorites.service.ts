/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Favorite, FavoriteCreate, FavoriteUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * One's favorites in a workspace (M7 design 3.8, M7/P7 spec 2.1): of a project, a cycle, a module or a view, a
 * project's or a workspace's, or a folder that holds such favorites, one level; only their owner reads and writes them,
 * while he is the workspace's admin or member. Its answers are nerve's, as nerve gives them.
 */
export class FavoritesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /**
   * The caller's favorites in the workspace, those his folders hold too, the greatest sequence first, then by id, all
   * at once; each of a target with the target's name and icon now. nerve refuses the workspace's guests (forbidden).
   */
  async list(slug: string): Promise<Favorite[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/favorites", { params: { path: { slug } } })).data;
  }

  /**
   * Favors a target, or makes a folder, in the workspace: the new favorite (201), or the caller's favorite of the
   * target he had already, as it is (200).
   */
  async create(slug: string, body: FavoriteCreate): Promise<Favorite> {
    return unwrap(await this.api.POST("/api/v0/workspaces/{slug}/favorites", { params: { path: { slug } }, body }));
  }

  /** Replaces each field body names; the answer is the favorite as nerve holds it, a change of nothing's too. */
  async update(favoriteId: string, body: FavoriteUpdate): Promise<Favorite> {
    return unwrap(
      await this.api.PATCH("/api/v0/favorites/{favorite_id}", {
        params: { path: { favorite_id: favoriteId } },
        body,
      })
    );
  }

  /** Deletes the favorite, a folder with every favorite it holds. */
  async remove(favoriteId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/favorites/{favorite_id}", { params: { path: { favorite_id: favoriteId } } }));
  }
}
