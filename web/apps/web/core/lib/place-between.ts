/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * The place that puts an item of a list ordered by a number (a sort order, a sequence: the lowest first) between its
 * new neighbours, given by their places: halfway between two; a step before the first, or past the last, at an end of
 * the list; none with no neighbour. A neighbour without a place (null) counts as none. The step is nerve's for the
 * list, its gap between the last item and a new one (M3 design 3.16-3.18, 4.10). A store reckons it in the change's
 * turn, from the list as nerve last answered it (v0 design 7.7).
 */
export function placeBetween(
  before: number | null | undefined,
  after: number | null | undefined,
  step: number
): number | undefined {
  if (before === undefined || before === null) return after === undefined || after === null ? undefined : after - step;
  return after === undefined || after === null ? before + step : (before + after) / 2;
}
