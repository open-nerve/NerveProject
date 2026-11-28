/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The favorites of the tests of the favorites store (M7/P7 spec 2.1), against a fake nerve: the caller's workspaces,
// acme and beta; his favorites as nerve lists them, of a target or a folder; the store, and its read.

import type { Favorite } from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import type { WorkspaceRef } from "@/store/favorites/favorite-changes";
import type { TargetKind } from "@/store/favorites/favorite-tree";
import { FavoriteStore } from "@/store/favorites/favorite.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

export const acme = workspaceOf("acme");
export const beta = workspaceOf("beta");

/** The requests of the favorites: a workspace's, listed and created there; a favorite's, changed and deleted. */
export const listOf = (workspace: { slug: string }) => `/api/v0/workspaces/${workspace.slug}/favorites`;
export const at = (favorite: Favorite) => `/api/v0/favorites/${favorite.id}`;

/**
 * The caller's favorite of a target in acme as nerve lists it, unless fields say: named by the target's id; of the
 * target's project, none for a workspace's view.
 */
export function favoriteOfTarget(
  kind: TargetKind,
  entityId: string,
  projectId: string | null,
  fields: Partial<Favorite> = {}
): Favorite {
  return {
    id: `f-${kind}-${entityId}`,
    workspace_id: acme.id,
    project_id: projectId,
    entity_type: kind,
    entity_id: entityId,
    parent_id: null,
    sequence: 65535,
    name: entityId,
    logo_props: {},
    created_at: "2026-11-01T09:00:00Z",
    ...fields,
  };
}

/** The caller's folder in acme as nerve lists it, unless fields say: of no project, at the top. */
export function favoriteFolder(name: string, fields: Partial<Favorite> = {}): Favorite {
  return {
    id: `f-folder-${name}`,
    workspace_id: acme.id,
    project_id: null,
    entity_type: "folder",
    entity_id: null,
    parent_id: null,
    sequence: 65535,
    name,
    logo_props: {},
    created_at: "2026-11-01T09:00:00Z",
    ...fields,
  };
}

/** A tab's favorites store, which sends through a fake nerve. */
export function favoritesTab() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  return { nerve, api, store: new FavoriteStore(api) };
}

/** The store reads the workspace's favorites, and nerve lists these. */
export function loadFavorites(
  nerve: FakeNerve,
  store: FavoriteStore,
  favorites: Favorite[],
  workspace: WorkspaceRef = acme
) {
  return answered(
    nerve,
    () => store.fetchFavorites(workspace),
    ["GET", listOf(workspace)],
    { data: favorites },
    "the favorites"
  );
}
