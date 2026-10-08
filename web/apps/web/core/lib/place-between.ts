/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * The place that puts an item of a list ordered by a number (a sort order, a sequence: the lowest first) between its
 * new neighbours, given by their places: halfway between two; a step before the first, or past the last, at an end of
 * the list; none with no neighbour. A neighbour without a place (null) counts as none. The step is nerve's for the
 * list, its gap between the last item and a new one (M3 design 3.16-3.18, 4.10).
 */
function placeBetween(before: number | null | undefined, after: number | null | undefined, step: number) {
  if (before === undefined || before === null) return after === undefined || after === null ? undefined : after - step;
  return after === undefined || after === null ? before + step : (before + after) / 2;
}

/**
 * The place of an item dropped among the items of a list, in their order, each placed by its key: before the item
 * droppedOnId names, or after it, or past the last at the end or for none (placeBetween). The project sidebar's moves,
 * the states' and the labels' reckon it so, in the change's turn, from the list as nerve last answered it (v0 design
 * 7.7).
 */
export function placeAt<K extends string, T extends { id: string } & Record<K, number | null>>(
  items: T[],
  key: K,
  droppedOnId: string | undefined,
  at: "before" | "after" | "end",
  step: number
): number | undefined {
  const droppedOn = items.findIndex((item) => item.id === droppedOnId);
  const index = at === "end" || droppedOn === -1 ? items.length : droppedOn + (at === "after" ? 1 : 0);
  return placeBetween(items[index - 1]?.[key], items[index]?.[key], step);
}
