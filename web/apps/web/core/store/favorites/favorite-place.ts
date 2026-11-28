/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Favorite, FavoriteUpdate } from "@nerve/api-client";
import { siblingsOf } from "./favorite-tree";

/**
 * Where a favorite is dropped in the sidebar (M7/P7 spec 2.4): on a row's top or bottom half, above or below it; on a
 * folder row's middle, into the folder; on the section's header, out of its folder.
 */
export type FavoriteDrop = { at: "above" | "below"; rowId: string } | { at: "into"; folderId: string } | { at: "out" };

/** The step past the first or short of the last: nerve's own between a new favorite and the greatest (createFavorite). */
const STEP = 10000;

/**
 * The sequence of a favorite dropped above or below the sibling droppedOnId, among siblings in the order the sidebar
 * shows them, the greatest first, the dropped one left out (M7/P7 spec 2.1, R4; D40): above it, halfway to the one
 * above, or STEP past it when it is the first; below it, halfway to the one below, or STEP short of it when it is the
 * last. 0 and negatives are sequences as any other. Two neighbours of the same sequence give theirs: the id orders
 * them then (favorite-tree.ts), and nerve renumbers none. Nothing when droppedOnId is not among the siblings.
 */
export function placeAmong(
  siblings: readonly Pick<Favorite, "id" | "sequence">[],
  droppedOnId: string,
  at: "above" | "below"
): number | undefined {
  const index = siblings.findIndex((sibling) => sibling.id === droppedOnId);
  const row = siblings[index];
  if (!row) return undefined;
  const neighbour = siblings[at === "above" ? index - 1 : index + 1]?.sequence;
  if (neighbour === undefined) return at === "above" ? row.sequence + STEP : row.sequence - STEP;
  return (row.sequence + neighbour) / 2;
}

/**
 * Whether the dragged favorite is right next to the row on the side it is dropped, among the siblings both are of:
 * dropped there, it stays where it is. Decided by place, not by the sequence a drop would reckon, which differs once
 * the dragged one is left out of the siblings (M7/P7 spec 2.1, re-98 m9).
 */
const besideAlready = (siblings: readonly Favorite[], draggedId: string, rowId: string, at: "above" | "below") => {
  const from = siblings.findIndex((sibling) => sibling.id === draggedId);
  const to = siblings.findIndex((sibling) => sibling.id === rowId);
  return from === (at === "above" ? to - 1 : to + 1);
};

/**
 * The change of the dragged favorite a drop makes, decided on the list as nerve answered it last, in the move's turn
 * (M7/P7 spec 2.1 move, 2.4): its folder (parent_id) when that changes, and its place among its new siblings
 * (placeAmong) when it is dropped above or below a row; into a folder or out of one, its sequence kept. Nothing when
 * nothing would move: the dragged favorite, the row or the folder gone from the list; a folder into a folder, or among
 * a folder's favorites (one level: dropped below a row only moved into a folder while the move was queued too, which
 * nerve would refuse); a drop on what is no folder; the place it has already (right next to the row on that side, in
 * the folder that holds it, out while at the top).
 */
export function moveOf(list: readonly Favorite[], draggedId: string, drop: FavoriteDrop): FavoriteUpdate | undefined {
  const dragged = list.find((favorite) => favorite.id === draggedId);
  if (!dragged) return undefined;
  const isFolder = dragged.entity_type === "folder";
  if (drop.at === "out") return dragged.parent_id === null ? undefined : { parent_id: null };
  if (drop.at === "into") {
    const folder = list.find((favorite) => favorite.id === drop.folderId);
    if (folder?.entity_type !== "folder" || isFolder || dragged.parent_id === folder.id) return undefined;
    return { parent_id: folder.id };
  }
  const row = list.find((favorite) => favorite.id === drop.rowId);
  if (!row || row.id === dragged.id) return undefined;
  if (isFolder && row.parent_id !== null) return undefined;
  const parentId = row.parent_id;
  const siblings = siblingsOf(list, parentId);
  if (dragged.parent_id === parentId && besideAlready(siblings, dragged.id, row.id, drop.at)) return undefined;
  const sequence = placeAmong(
    siblings.filter((sibling) => sibling.id !== dragged.id),
    row.id,
    drop.at
  );
  if (sequence === undefined) return undefined;
  return dragged.parent_id === parentId ? { sequence } : { parent_id: parentId, sequence };
}
