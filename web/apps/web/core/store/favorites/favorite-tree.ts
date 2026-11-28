/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Favorite, FavoriteKind } from "@nerve/api-client";

/** What a favorite of a target is of; a view, a project's or a workspace's alike (the contract's view). */
export type TargetKind = Exclude<FavoriteKind, "folder">;

/** A favorite at the top of the tree, with the favorites it holds in their order: a folder's; none for the others. */
export type FavoriteBranch = { favorite: Favorite; children: Favorite[] };

/**
 * The order nerve lists favorites in (listFavorites): the greatest sequence first, then by id, both descending. Two of
 * the same sequence (a drop between equal neighbours) are ordered by id; nerve renumbers none.
 */
const inOrder = (a: Favorite, b: Favorite): number =>
  b.sequence - a.sequence || (a.id < b.id ? 1 : a.id > b.id ? -1 : 0);

/** The favorites the folder holds (parentId), or those at the top (null), in their order. */
export function siblingsOf(list: readonly Favorite[], parentId: string | null): Favorite[] {
  return list.filter((favorite) => favorite.parent_id === parentId).toSorted(inOrder);
}

/**
 * The sidebar's tree of one's favorites in a workspace (M7/P7 spec 2.1): those at the top, each folder with the
 * favorites it holds under it, each level in nerve's order, whatever the order of the list. A favorite whose folder the
 * list does not hold shows nowhere: nerve lists none such (a folder goes with what it holds), so it guards nothing
 * nerve answers but a list changed here out of step.
 */
export function favoriteTree(list: readonly Favorite[]): FavoriteBranch[] {
  return siblingsOf(list, null).map((favorite) => ({
    favorite,
    children: favorite.entity_type === "folder" ? siblingsOf(list, favorite.id) : [],
  }));
}

/** The favorite of the target in the list (M7/P7 spec 2.1): of its kind and its id; a view's of either container. */
export function favoriteIn(list: readonly Favorite[], kind: TargetKind, entityId: string): Favorite | undefined {
  return list.find((favorite) => favorite.entity_type === kind && favorite.entity_id === entityId);
}
