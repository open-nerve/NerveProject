/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * The command palette's filter, both hosts' (wrapper.tsx, top-nav-power-k.tsx): the search finds an item by the text
 * it gives, case aside: its keywords when it gives them, its value otherwise. cmdk keeps a list's selection by its
 * items' values, so a row whose text another may share (two cycles of one name, a cycle named as No cycle) is valued
 * by its id and found by its keywords, its value never searched (P7c review re-77 M1). The no-results row always shows.
 */
export function powerKFilter(value: string, search: string, keywords: string[] = []): number {
  if (value === "no-results") return 1;
  const term = search.toLowerCase();
  return (keywords.length > 0 ? keywords : [value]).some((text) => text.toLowerCase().includes(term)) ? 1 : 0;
}
